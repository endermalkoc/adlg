package workspace

import (
	"context"
	"database/sql"
)

// TableDiff is the per-table row-change summary between two refs.
type TableDiff struct {
	Table    string `json:"table"`
	Added    int64  `json:"added"`
	Modified int64  `json:"modified"`
	Deleted  int64  `json:"deleted"`
}

// Diff returns the per-table row changes between two refs (commits or branches)
// using Dolt's dolt_diff_stat table function — the PR-style summary that lets a
// changeset's edits across many tables be reviewed together. Runs on the pinned
// connection.
func Diff(ctx context.Context, conn *sql.Conn, from, to string) ([]TableDiff, error) {
	rows, err := conn.QueryContext(ctx,
		"SELECT table_name, rows_added, rows_modified, rows_deleted FROM dolt_diff_stat(?, ?)", from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TableDiff
	for rows.Next() {
		var d TableDiff
		if err := rows.Scan(&d.Table, &d.Added, &d.Modified, &d.Deleted); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
