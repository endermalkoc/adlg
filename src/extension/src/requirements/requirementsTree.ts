import * as vscode from "vscode";
import { CuspClient, DomainNode, SpecNode, ReqNode } from "../cusp/client";

type Kind = "domain" | "spec" | "category" | "story" | "group" | "req" | "section";

// A node in a STABLE tree built once per load. Stable object identity + a stable `id` are what
// let TreeView.reveal() walk from an element up through getParent() and expand/select it.
interface Node {
  kind: Kind;
  id: string;
  label: string;
  description?: string;
  tooltip?: string;
  icon?: string;
  docPath?: string; // spec / story / req / section — all open the spec doc
  anchor?: string; // req: frKey.toLowerCase() (scroll to an id)
  find?: string; // story / section: heading text to scroll to (scroll by text)
  openTitle?: string; // webview panel title when opened
  parent?: Node;
  children: Node[];
}

export class RequirementsTreeProvider implements vscode.TreeDataProvider<Node> {
  private readonly _onDidChangeTreeData = new vscode.EventEmitter<void>();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;
  private roots: Node[] | undefined;
  private loading: Promise<Node[]> | undefined;

  constructor(private client: CuspClient) {}

  setClient(client: CuspClient): void {
    this.client = client;
  }

  refresh(): void {
    this.roots = undefined;
    this.loading = undefined;
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(node: Node): vscode.TreeItem {
    const leaf = node.kind === "req" || node.kind === "story" || node.kind === "section";
    const item = new vscode.TreeItem(
      node.label,
      leaf ? vscode.TreeItemCollapsibleState.None : vscode.TreeItemCollapsibleState.Collapsed,
    );
    item.id = node.id;
    item.description = node.description;
    if (node.tooltip) {
      item.tooltip = node.kind === "req" ? new vscode.MarkdownString(node.tooltip) : node.tooltip;
    }
    item.contextValue = `cusp-${node.kind}`;
    item.iconPath = new vscode.ThemeIcon(node.icon ?? iconFor(node.kind));
    if (node.docPath) {
      // Specs open at the top; FRs scroll to their anchor; stories/sections scroll to their heading.
      item.command = {
        command: "cusp.openSpecDoc",
        title: "Open",
        arguments: [{ docPath: node.docPath, anchor: node.anchor, find: node.find, title: node.openTitle }],
      };
    }
    return item;
  }

  getChildren(node?: Node): Node[] | Promise<Node[]> {
    return node ? node.children : this.ensureRoots();
  }

  getParent(node: Node): Node | undefined {
    return node.parent;
  }

  // find locates a spec (by doc path) or, when an anchor is given, the FR under it (searched
  // recursively, since FRs now sit under the "Functional Requirements" category and its groups) —
  // the element handed to TreeView.reveal so a link/back-forward navigation follows in the tree.
  async find(docPath: string, anchor?: string): Promise<Node | undefined> {
    const roots = await this.ensureRoots();
    for (const domain of roots) {
      for (const spec of domain.children) {
        if (spec.docPath !== docPath) {
          continue;
        }
        return anchor ? findByAnchor(spec, anchor) ?? spec : spec;
      }
    }
    return undefined;
  }

  private ensureRoots(): Promise<Node[]> {
    if (this.roots) {
      return Promise.resolve(this.roots);
    }
    if (!this.loading) {
      this.loading = this.client
        .requirementsTree()
        .then((domains) => (this.roots = (domains ?? []).map(buildDomain)))
        .catch((err) => {
          vscode.window.showErrorMessage(`Cusp: failed to load requirements — ${messageOf(err)}`);
          this.loading = undefined;
          return [];
        });
    }
    return this.loading;
  }
}

function buildDomain(d: DomainNode): Node {
  const domain: Node = { kind: "domain", id: `domain:${d.slug}`, label: d.name, children: [] };
  domain.children = (d.specs ?? []).map((s) => buildSpec(s, domain));
  return domain;
}

function buildSpec(s: SpecNode, domain: Node): Node {
  const specTitle = s.title || s.prefix || s.docPath;
  const spec: Node = {
    kind: "spec",
    id: `spec:${s.docPath}`,
    label: specTitle, // "Add Student [ADDS]" — prefix bracketed in the description
    description: s.title && s.prefix ? `[${s.prefix}]` : undefined,
    tooltip: s.docPath,
    docPath: s.docPath,
    openTitle: specTitle,
    parent: domain,
    children: [],
  };

  const cats: Node[] = [];

  // User Stories.
  const stories = s.stories ?? [];
  if (stories.length) {
    const cat = categoryNode("User Stories", `cat:${s.docPath}:stories`, "account", spec);
    cat.children = stories.map((st, i) => ({
      kind: "story" as Kind,
      id: `story:${s.docPath}:${i}`,
      label: st.title,
      docPath: s.docPath,
      find: st.title, // scroll to the "User Story N - <title> …" heading
      openTitle: `${st.title} — ${specTitle}`,
      parent: cat,
      children: [],
    }));
    cats.push(cat);
  }

  // Functional Requirements: groups (each with its FRs) + ungrouped FRs.
  const groups = s.groups ?? [];
  const ungrouped = s.requirements ?? [];
  if (groups.length || ungrouped.length) {
    const cat = categoryNode("Functional Requirements", `cat:${s.docPath}:frs`, "checklist", spec);
    const mkReq = (r: ReqNode, parent: Node): Node => ({
      kind: "req",
      id: `req:${r.frKey}`,
      label: r.frKey,
      description: r.statement,
      tooltip: `**${r.frKey}**\n\n${r.statement}`,
      docPath: s.docPath,
      anchor: r.frKey.toLowerCase(),
      openTitle: `${r.frKey} — ${specTitle}`,
      parent,
      children: [],
    });
    const groupNodes: Node[] = groups.map((g) => {
      const gn: Node = {
        kind: "group",
        id: `group:${s.docPath}#${g.title}`,
        label: g.title,
        parent: cat,
        children: [],
      };
      gn.children = (g.requirements ?? []).map((r) => mkReq(r, gn));
      return gn;
    });
    cat.children = [...groupNodes, ...ungrouped.map((r) => mkReq(r, cat))];
    cats.push(cat);
  }

  // Other: prose sections.
  const sections = s.sections ?? [];
  if (sections.length) {
    const cat = categoryNode("Other", `cat:${s.docPath}:other`, "note", spec);
    cat.children = sections.map((sec) => ({
      kind: "section" as Kind,
      id: `section:${s.docPath}:${sec.key}`,
      label: sec.title,
      docPath: s.docPath,
      find: sec.title, // scroll to the section heading
      openTitle: `${sec.title} — ${specTitle}`,
      parent: cat,
      children: [],
    }));
    cats.push(cat);
  }

  spec.children = cats;
  return spec;
}

function categoryNode(label: string, id: string, icon: string, parent: Node): Node {
  return { kind: "category", id, label, icon, parent, children: [] };
}

function findByAnchor(node: Node, anchor: string): Node | undefined {
  if (node.anchor === anchor) {
    return node;
  }
  for (const child of node.children) {
    const hit = findByAnchor(child, anchor);
    if (hit) {
      return hit;
    }
  }
  return undefined;
}

function iconFor(kind: Kind): string {
  switch (kind) {
    case "domain":
      return "folder";
    case "spec":
      return "file";
    case "category":
      return "list-tree";
    case "story":
      return "person";
    case "group":
      return "symbol-namespace";
    case "section":
      return "note";
    default:
      return "symbol-field"; // req
  }
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
