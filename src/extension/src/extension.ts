import * as vscode from "vscode";
import { CliCuspClient } from "./cusp/cliClient";
import { ChangesetTreeProvider } from "./changesets/changesetTree";
import { RequirementsTreeProvider } from "./requirements/requirementsTree";
import { registerSpecDocView } from "./requirements/specDocView";

export function activate(context: vscode.ExtensionContext): void {
  const makeClient = () => new CliCuspClient(resolveCliPath(), resolveWorkspaceDir());
  let client = makeClient();

  const changesets = new ChangesetTreeProvider(client);
  const requirements = new RequirementsTreeProvider(client);
  // A TreeView (not just a provider) so we can reveal() — expand + select the node the doc panel
  // navigates to (link clicks and back/forward included) — and get a built-in Collapse All button.
  const reqView = vscode.window.createTreeView("cuspRequirements", {
    treeDataProvider: requirements,
    showCollapseAll: true,
  });

  // Reflect the active filter in the view (a "Filter: …" subtitle + a context key that shows the
  // Clear button).
  const applyFilterUI = (q: string) => {
    reqView.description = q ? `Filter: ${q}` : undefined;
    void vscode.commands.executeCommand("setContext", "cusp.requirementsFiltered", q.length > 0);
  };

  const revealInTree = async (docPath: string, anchor?: string) => {
    const node = await requirements.find(docPath, anchor);
    if (node) {
      try {
        await reqView.reveal(node, { select: true, focus: false, expand: true });
      } catch {
        /* the view may be hidden — reveal is best-effort */
      }
    }
  };

  context.subscriptions.push(
    vscode.window.registerTreeDataProvider("cuspChangesets", changesets),
    reqView,
    vscode.commands.registerCommand("cusp.refreshChangesets", () => changesets.refresh()),
    vscode.commands.registerCommand("cusp.refreshRequirements", () => requirements.refresh()),
    vscode.commands.registerCommand("cusp.searchRequirements", async () => {
      const q = await vscode.window.showInputBox({
        prompt: "Filter requirements — matches FR keys, statements, and spec/story/section titles",
        placeHolder: "e.g. email, invoice, ADDS-FR",
        value: requirements.filterText,
      });
      if (q === undefined) {
        return; // cancelled — leave the current filter unchanged
      }
      requirements.setFilter(q);
      applyFilterUI(q.trim());
    }),
    vscode.commands.registerCommand("cusp.clearRequirementsSearch", () => {
      requirements.setFilter("");
      applyFilterUI("");
    }),
    registerSpecDocView(() => client, revealInTree),
    // Rebuild the transport when the relevant settings change — no reload needed.
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("cusp.cliPath") || e.affectsConfiguration("cusp.workspaceFolder")) {
        client = makeClient();
        changesets.setClient(client);
        requirements.setClient(client);
        changesets.refresh();
        requirements.refresh();
      }
    }),
  );
}

export function deactivate(): void {
  // Nothing to tear down — the extension holds no state; Dolt is the source of truth.
}

function resolveCliPath(): string {
  return vscode.workspace.getConfiguration("cusp").get<string>("cliPath")?.trim() || "cusp";
}

function resolveWorkspaceDir(): string {
  const configured = vscode.workspace.getConfiguration("cusp").get<string>("workspaceFolder")?.trim();
  if (configured) {
    return configured;
  }
  const folders = vscode.workspace.workspaceFolders;
  if (folders && folders.length > 0) {
    return folders[0].uri.fsPath;
  }
  return process.cwd();
}
