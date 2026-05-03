package html

// reportTemplate is the complete HTML template with embedded CSS and JavaScript.
const reportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DeepDiff DB Report</title>
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        :root {
            --bg: #ffffff;
            --bg-secondary: #f9fafb;
            --bg-tertiary: #f3f4f6;
            --border: #e5e7eb;
            --border-light: #f3f4f6;
            --text: #111827;
            --text-secondary: #6b7280;
            --text-tertiary: #9ca3af;
            --accent: #2563eb;
            --accent-light: #eff6ff;
            --success: #059669;
            --success-light: #ecfdf5;
            --warning: #d97706;
            --warning-light: #fffbeb;
            --danger: #dc2626;
            --danger-light: #fef2f2;
            --font-mono: 'SF Mono', SFMono-Regular, ui-monospace, Menlo, Monaco, Consolas, monospace;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: var(--bg);
            color: var(--text);
            font-size: 14px;
            line-height: 1.5;
            -webkit-font-smoothing: antialiased;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 32px 24px;
        }

        /* Header */
        .header {
            margin-bottom: 32px;
            padding-bottom: 24px;
            border-bottom: 1px solid var(--border);
        }

        .header-top {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 16px;
        }

        .header h1 {
            font-size: 20px;
            font-weight: 600;
            color: var(--text);
        }

        .header-actions {
            display: flex;
            gap: 8px;
        }

        .btn {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            padding: 8px 14px;
            font-size: 13px;
            font-weight: 500;
            border-radius: 6px;
            border: 1px solid var(--border);
            background: var(--bg);
            color: var(--text-secondary);
            cursor: pointer;
            transition: all 0.15s;
        }

        .btn:hover {
            background: var(--bg-secondary);
            color: var(--text);
            border-color: var(--text-tertiary);
        }

        .btn-primary {
            background: var(--text);
            color: var(--bg);
            border-color: var(--text);
        }

        .btn-primary:hover {
            background: #374151;
            border-color: #374151;
            color: var(--bg);
        }

        .meta {
            display: flex;
            flex-wrap: wrap;
            gap: 24px;
            font-size: 13px;
            color: var(--text-secondary);
        }

        .meta-item {
            display: flex;
            align-items: center;
            gap: 6px;
        }

        .meta-label {
            color: var(--text-tertiary);
        }

        /* Stats */
        .stats {
            display: grid;
            grid-template-columns: repeat(6, 1fr);
            gap: 1px;
            background: var(--border);
            border: 1px solid var(--border);
            border-radius: 8px;
            overflow: hidden;
            margin-bottom: 32px;
        }

        .stat {
            background: var(--bg);
            padding: 16px 20px;
            text-align: center;
        }

        .stat-value {
            font-size: 24px;
            font-weight: 600;
            color: var(--text);
            font-variant-numeric: tabular-nums;
        }

        .stat-value.success { color: var(--success); }
        .stat-value.warning { color: var(--warning); }
        .stat-value.danger { color: var(--danger); }

        .stat-label {
            font-size: 12px;
            color: var(--text-tertiary);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-top: 4px;
        }

        /* Tabs */
        .tabs {
            border: 1px solid var(--border);
            border-radius: 8px;
            overflow: hidden;
        }

        .tab-nav {
            display: flex;
            background: var(--bg-secondary);
            border-bottom: 1px solid var(--border);
            padding: 0 4px;
        }

        .tab-btn {
            padding: 12px 16px;
            font-size: 13px;
            font-weight: 500;
            color: var(--text-secondary);
            background: transparent;
            border: none;
            cursor: pointer;
            position: relative;
            transition: color 0.15s;
        }

        .tab-btn:hover {
            color: var(--text);
        }

        .tab-btn.active {
            color: var(--text);
        }

        .tab-btn.active::after {
            content: '';
            position: absolute;
            bottom: -1px;
            left: 0;
            right: 0;
            height: 2px;
            background: var(--text);
        }

        .tab-count {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            min-width: 18px;
            height: 18px;
            padding: 0 5px;
            margin-left: 6px;
            font-size: 11px;
            font-weight: 500;
            border-radius: 9px;
            background: var(--bg-tertiary);
            color: var(--text-secondary);
        }

        .tab-count.has-items { background: var(--accent-light); color: var(--accent); }
        .tab-count.has-warning { background: var(--warning-light); color: var(--warning); }
        .tab-count.has-danger { background: var(--danger-light); color: var(--danger); }

        .tab-panel {
            display: none;
            padding: 20px;
        }

        .tab-panel.active {
            display: block;
        }

        /* Tables */
        .table-wrap {
            overflow-x: auto;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            font-size: 13px;
        }

        th {
            text-align: left;
            padding: 10px 12px;
            font-weight: 500;
            font-size: 11px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            color: var(--text-tertiary);
            background: var(--bg-secondary);
            border-bottom: 1px solid var(--border);
        }

        td {
            padding: 12px;
            border-bottom: 1px solid var(--border-light);
            vertical-align: top;
        }

        tr:last-child td {
            border-bottom: none;
        }

        tr:hover td {
            background: var(--bg-secondary);
        }

        .mono {
            font-family: var(--font-mono);
            font-size: 12px;
        }

        /* Badges */
        .badge {
            display: inline-flex;
            align-items: center;
            padding: 2px 8px;
            font-size: 11px;
            font-weight: 500;
            border-radius: 4px;
            text-transform: uppercase;
            letter-spacing: 0.3px;
        }

        .badge-success { background: var(--success-light); color: var(--success); }
        .badge-warning { background: var(--warning-light); color: var(--warning); }
        .badge-danger { background: var(--danger-light); color: var(--danger); }
        .badge-neutral { background: var(--bg-tertiary); color: var(--text-secondary); }

        /* Resolution Summary */
        .resolution-summary {
            margin-bottom: 24px;
            padding: 20px;
            background: var(--bg-secondary);
            border: 1px solid var(--border);
            border-radius: 8px;
        }

        .section-title {
            font-size: 14px;
            font-weight: 600;
            color: var(--text);
            margin-bottom: 16px;
        }

        .resolution-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 16px;
            margin-bottom: 20px;
        }

        .resolution-stat {
            text-align: center;
            padding: 16px;
            background: var(--bg);
            border: 1px solid var(--border);
            border-radius: 6px;
        }

        .resolution-value {
            font-size: 28px;
            font-weight: 600;
            font-variant-numeric: tabular-nums;
            margin-bottom: 4px;
        }

        .resolution-value.success { color: var(--success); }
        .resolution-value.warning { color: var(--warning); }
        .resolution-value.neutral { color: var(--text-secondary); }

        .resolution-label {
            font-size: 12px;
            color: var(--text-secondary);
        }

        .strategy-table-wrap {
            overflow-x: auto;
        }

        .strategy-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 13px;
        }

        .strategy-table th,
        .strategy-table td {
            padding: 10px 12px;
            text-align: left;
            border-bottom: 1px solid var(--border);
        }

        .strategy-table th {
            font-weight: 500;
            color: var(--text-secondary);
            background: var(--bg);
        }

        .strategy-table td:nth-child(n+3) {
            text-align: center;
        }

        .strategy-table th:nth-child(n+3) {
            text-align: center;
        }

        /* Expandable keys */
        .data-row {
            cursor: pointer;
        }

        .data-row:hover {
            background: var(--bg-secondary);
        }

        .expand-hint {
            font-size: 11px;
            color: var(--text-tertiary);
            margin-left: 8px;
        }

        .keys-row td {
            padding: 0 !important;
            background: var(--bg-secondary);
        }

        .keys-detail {
            padding: 12px 16px;
        }

        .keys-section {
            margin-bottom: 10px;
        }

        .keys-section:last-child {
            margin-bottom: 0;
        }

        .keys-label {
            display: block;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.3px;
            margin-bottom: 6px;
        }

        .keys-list {
            display: flex;
            flex-wrap: wrap;
            gap: 6px;
        }

        .key-item {
            display: inline-block;
            padding: 3px 8px;
            background: var(--bg);
            border: 1px solid var(--border);
            border-radius: 4px;
            font-family: var(--font-mono);
            font-size: 11px;
        }

        /* Conflict badges */
        .conflict-badges {
            display: flex;
            gap: 6px;
            align-items: center;
        }

        .badge-strategy {
            font-size: 10px;
        }

        /* Change indicators */
        .change {
            display: inline-flex;
            align-items: center;
            gap: 4px;
            font-family: var(--font-mono);
            font-size: 12px;
        }

        .change-add { color: var(--success); }
        .change-remove { color: var(--danger); }
        .change-modify { color: var(--warning); }

        /* Schema diff */
        .diff-section {
            margin-bottom: 16px;
        }

        .diff-header {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px 16px;
            background: var(--bg-secondary);
            border: 1px solid var(--border);
            border-radius: 6px;
            cursor: pointer;
            transition: background 0.15s;
        }

        .diff-header:hover {
            background: var(--bg-tertiary);
        }

        .diff-header.open {
            border-radius: 6px 6px 0 0;
            border-bottom-color: transparent;
        }

        .diff-chevron {
            width: 16px;
            height: 16px;
            color: var(--text-tertiary);
            transition: transform 0.15s;
        }

        .diff-header.open .diff-chevron {
            transform: rotate(90deg);
        }

        .diff-table {
            font-family: var(--font-mono);
            font-size: 13px;
            font-weight: 500;
        }

        .diff-type {
            margin-left: auto;
            font-size: 11px;
        }

        .diff-body {
            display: none;
            border: 1px solid var(--border);
            border-top: none;
            border-radius: 0 0 6px 6px;
            overflow: hidden;
        }

        .diff-header.open + .diff-body {
            display: block;
        }

        .diff-row {
            display: flex;
            align-items: stretch;
            border-bottom: 1px solid var(--border-light);
            font-size: 13px;
        }

        .diff-row:last-child {
            border-bottom: none;
        }

        .diff-indicator {
            width: 32px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-family: var(--font-mono);
            font-size: 12px;
            font-weight: 600;
            flex-shrink: 0;
        }

        .diff-row.add .diff-indicator { background: var(--success-light); color: var(--success); }
        .diff-row.remove .diff-indicator { background: var(--danger-light); color: var(--danger); }
        .diff-row.modify .diff-indicator { background: var(--warning-light); color: var(--warning); }

        .diff-content {
            flex: 1;
            padding: 10px 14px;
            font-family: var(--font-mono);
            font-size: 12px;
        }

        .diff-row.add .diff-content { background: #f0fdf4; }
        .diff-row.remove .diff-content { background: #fef2f2; }
        .diff-row.modify .diff-content { background: #fffbeb; }

        .diff-col {
            color: var(--text);
        }

        .diff-detail {
            color: var(--text-secondary);
            margin-left: 8px;
        }

        /* Filters */
        .filters {
            display: flex;
            gap: 12px;
            margin-bottom: 16px;
        }

        .filter-select {
            padding: 7px 10px;
            font-size: 13px;
            border: 1px solid var(--border);
            border-radius: 6px;
            background: var(--bg);
            color: var(--text);
            min-width: 160px;
        }

        .filter-select:focus {
            outline: none;
            border-color: var(--accent);
            box-shadow: 0 0 0 3px var(--accent-light);
        }

        /* Conflict list */
        .conflict-list {
            display: flex;
            flex-direction: column;
            gap: 1px;
            background: var(--border);
            border: 1px solid var(--border);
            border-radius: 6px;
            overflow: hidden;
        }

        .conflict-item {
            display: grid;
            grid-template-columns: 140px 1fr 200px 100px;
            gap: 16px;
            align-items: center;
            padding: 12px 16px;
            background: var(--bg);
            font-size: 13px;
        }

        .conflict-item:hover {
            background: var(--bg-secondary);
        }

        .conflict-table {
            font-family: var(--font-mono);
            font-weight: 500;
        }

        .conflict-key {
            font-family: var(--font-mono);
            font-size: 12px;
            color: var(--text-secondary);
        }

        .conflict-hashes {
            display: flex;
            align-items: center;
            gap: 8px;
            font-family: var(--font-mono);
            font-size: 11px;
        }

        .hash {
            padding: 3px 6px;
            background: var(--bg-tertiary);
            border-radius: 3px;
            color: var(--text-secondary);
        }

        .hash-arrow {
            color: var(--text-tertiary);
        }

        /* SQL Code */
        .sql-wrap {
            border: 1px solid var(--border);
            border-radius: 6px;
            overflow: hidden;
        }

        .sql-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 10px 14px;
            background: var(--bg-secondary);
            border-bottom: 1px solid var(--border);
        }

        .sql-title {
            font-size: 12px;
            font-weight: 500;
            color: var(--text-secondary);
        }

        .sql-code {
            background: #1e1e1e;
            padding: 16px;
            overflow-x: auto;
            max-height: 600px;
            overflow-y: auto;
        }

        .sql-code pre {
            margin: 0;
            font-family: var(--font-mono);
            font-size: 12px;
            line-height: 1.6;
            color: #d4d4d4;
            white-space: pre;
        }

        .sql-keyword { color: #569cd6; }
        .sql-string { color: #ce9178; }
        .sql-number { color: #b5cea8; }
        .sql-comment { color: #6a9955; }
        .sql-function { color: #dcdcaa; }

        /* Empty state */
        .empty {
            text-align: center;
            padding: 48px 24px;
            color: var(--text-secondary);
        }

        .empty-title {
            font-size: 14px;
            font-weight: 500;
            color: var(--text);
            margin-bottom: 4px;
        }

        .empty-desc {
            font-size: 13px;
        }

        /* Key list */
        .key-list {
            display: flex;
            flex-wrap: wrap;
            gap: 4px;
            margin-top: 8px;
        }

        .key-tag {
            font-family: var(--font-mono);
            font-size: 11px;
            padding: 2px 6px;
            background: var(--bg-tertiary);
            border-radius: 3px;
            color: var(--text-secondary);
        }

        /* Responsive */
        @media (max-width: 900px) {
            .stats {
                grid-template-columns: repeat(3, 1fr);
            }
            .conflict-item {
                grid-template-columns: 1fr;
                gap: 8px;
            }
        }

        @media (max-width: 600px) {
            .stats {
                grid-template-columns: repeat(2, 1fr);
            }
            .header-top {
                flex-direction: column;
                align-items: flex-start;
                gap: 12px;
            }
            .meta {
                flex-direction: column;
                gap: 8px;
            }
        }

        /* Print */
        @media print {
            body { font-size: 11px; }
            .container { padding: 0; max-width: none; }
            .btn, .filters { display: none !important; }
            .tab-panel { display: block !important; padding: 16px 0; border-bottom: 1px solid var(--border); }
            .tab-nav { display: none; }
            .tabs { border: none; }
            .tab-panel::before {
                content: attr(data-title);
                display: block;
                font-size: 14px;
                font-weight: 600;
                margin-bottom: 12px;
                padding-bottom: 8px;
                border-bottom: 1px solid var(--border);
            }
            .diff-body { display: block !important; }
            .sql-code { max-height: none; background: #f9fafb !important; }
            .sql-code pre { color: #111827 !important; }
        }
    </style>
</head>
<body>
    <div class="container">
        <header class="header">
            <div class="header-top">
                <h1>Database Diff Report</h1>
                <div class="header-actions">
                    <button class="btn btn-primary" onclick="window.print()">Export PDF</button>
                </div>
            </div>
            <div class="meta">
                <div class="meta-item">
                    <span class="meta-label">Generated</span>
                    <span>{{formatTime .GeneratedAt}}</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Source</span>
                    <span class="mono">{{.ProdDB}}</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Target</span>
                    <span class="mono">{{.DevDB}}</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Version</span>
                    <span>{{.Version}}</span>
                </div>
            </div>
        </header>

        <div class="stats">
            <div class="stat">
                <div class="stat-value {{if eq .Summary.SchemaStatus "DRIFT"}}warning{{else}}success{{end}}">{{.Summary.SchemaStatus}}</div>
                <div class="stat-label">Schema</div>
            </div>
            <div class="stat">
                <div class="stat-value">{{.Summary.TablesScanned}}</div>
                <div class="stat-label">Tables</div>
            </div>
            <div class="stat">
                <div class="stat-value {{if gt .Summary.AddedRows 0}}success{{end}}">{{if gt .Summary.AddedRows 0}}+{{end}}{{.Summary.AddedRows}}</div>
                <div class="stat-label">Added</div>
            </div>
            <div class="stat">
                <div class="stat-value {{if gt .Summary.RemovedRows 0}}danger{{end}}">{{if gt .Summary.RemovedRows 0}}-{{end}}{{.Summary.RemovedRows}}</div>
                <div class="stat-label">Removed</div>
            </div>
            <div class="stat">
                <div class="stat-value {{if gt .Summary.UpdatedRows 0}}warning{{end}}">{{.Summary.UpdatedRows}}</div>
                <div class="stat-label">Modified</div>
            </div>
            <div class="stat">
                <div class="stat-value {{if gt .Summary.TotalConflicts 0}}danger{{end}}">{{.Summary.TotalConflicts}}</div>
                <div class="stat-label">Conflicts</div>
            </div>
        </div>

        {{if .ResolutionBreakdown}}
        <div class="resolution-summary">
            <h3 class="section-title">Resolution Summary</h3>
            <div class="resolution-grid">
                <div class="resolution-stat">
                    <div class="resolution-value success">{{.ResolutionBreakdown.AutoResolvedTheirs}}</div>
                    <div class="resolution-label">Auto (use target)</div>
                </div>
                <div class="resolution-stat">
                    <div class="resolution-value neutral">{{.ResolutionBreakdown.AutoResolvedOurs}}</div>
                    <div class="resolution-label">Auto (keep source)</div>
                </div>
                <div class="resolution-stat">
                    <div class="resolution-value warning">{{.ResolutionBreakdown.PendingManual}}</div>
                    <div class="resolution-label">Pending review</div>
                </div>
            </div>
            {{if .ResolutionBreakdown.TableStrategies}}
            <div class="strategy-table-wrap">
                <table class="strategy-table">
                    <thead>
                        <tr>
                            <th>Table</th>
                            <th>Strategy</th>
                            <th>Conflicts</th>
                            <th>Resolved</th>
                            <th>Pending</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .ResolutionBreakdown.TableStrategies}}
                        <tr>
                            <td><span class="mono">{{.Table}}</span></td>
                            <td>
                                {{if eq .Strategy "ours"}}<span class="badge badge-neutral">ours</span>
                                {{else if eq .Strategy "theirs"}}<span class="badge badge-success">theirs</span>
                                {{else}}<span class="badge badge-warning">manual</span>{{end}}
                            </td>
                            <td>{{.ConflictCount}}</td>
                            <td>{{if gt .ResolvedCount 0}}<span class="change change-add">{{.ResolvedCount}}</span>{{else}}0{{end}}</td>
                            <td>{{if gt .PendingCount 0}}<span class="change change-modify">{{.PendingCount}}</span>{{else}}0{{end}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
            {{end}}
        </div>
        {{end}}

        <div class="tabs">
            <nav class="tab-nav">
                <button class="tab-btn active" data-tab="schema" onclick="switchTab('schema')">
                    Schema{{if .HasSchemaDiff}}<span class="tab-count has-warning">{{add (len .SchemaChanges) .Summary.ViewsChanged .Summary.RoutinesChanged .Summary.TriggersChanged .Summary.SequencesChanged}}</span>{{end}}
                </button>
                <button class="tab-btn" data-tab="data" onclick="switchTab('data')">
                    Data{{if .HasDataDiff}}<span class="tab-count has-items">{{.Summary.TablesWithChanges}}</span>{{end}}
                </button>
                <button class="tab-btn" data-tab="conflicts" onclick="switchTab('conflicts')">
                    Conflicts{{if .HasConflicts}}<span class="tab-count has-danger">{{.Summary.TotalConflicts}}</span>{{end}}
                </button>
                {{if .HasMigration}}
                <button class="tab-btn" data-tab="sql" onclick="switchTab('sql')">Migration</button>
                {{end}}
            </nav>

            <div id="tab-schema" class="tab-panel active" data-title="Schema Changes">
                {{if .HasSchemaDiff}}
                    {{range .SchemaChanges}}
                    <div class="diff-section">
                        <div class="diff-header" onclick="this.classList.toggle('open')">
                            <svg class="diff-chevron" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M6.22 3.22a.75.75 0 011.06 0l4.25 4.25a.75.75 0 010 1.06l-4.25 4.25a.75.75 0 01-1.06-1.06L9.94 8 6.22 4.28a.75.75 0 010-1.06z"/>
                            </svg>
                            <span class="diff-table">{{.Table}}</span>
                            <span class="diff-type">
                                {{if eq .ChangeType "added_table"}}<span class="badge badge-success">New Table</span>
                                {{else if eq .ChangeType "removed_table"}}<span class="badge badge-danger">Dropped</span>
                                {{else}}<span class="badge badge-warning">Modified</span>{{end}}
                            </span>
                        </div>
                        <div class="diff-body">
                            {{range .ColumnChanges}}
                            <div class="diff-row {{if eq .ChangeType "added"}}add{{else if eq .ChangeType "removed"}}remove{{else}}modify{{end}}">
                                <div class="diff-indicator">{{if eq .ChangeType "added"}}+{{else if eq .ChangeType "removed"}}−{{else}}~{{end}}</div>
                                <div class="diff-content">
                                    <span class="diff-col">{{.Column}}</span>
                                    <span class="diff-detail">{{.Description}}</span>
                                </div>
                            </div>
                            {{end}}
                            {{range .IndexChanges}}
                            <div class="diff-row {{if eq .ChangeType "added"}}add{{else if eq .ChangeType "removed"}}remove{{else}}modify{{end}}">
                                <div class="diff-indicator">{{if eq .ChangeType "added"}}+{{else if eq .ChangeType "removed"}}−{{else}}~{{end}}</div>
                                <div class="diff-content">
                                    <span class="diff-col">INDEX {{.Name}}</span>
                                    <span class="diff-detail">({{join .Columns ", "}})</span>
                                </div>
                            </div>
                            {{end}}
                            {{range .ForeignKeyChanges}}
                            <div class="diff-row {{if eq .ChangeType "added"}}add{{else if eq .ChangeType "removed"}}remove{{else}}modify{{end}}">
                                <div class="diff-indicator">{{if eq .ChangeType "added"}}+{{else if eq .ChangeType "removed"}}−{{else}}~{{end}}</div>
                                <div class="diff-content">
                                    <span class="diff-col">FK {{.Name}}</span>
                                    <span class="diff-detail">{{.Description}}</span>
                                </div>
                            </div>
                            {{end}}
                            {{if and (not .ColumnChanges) (not .IndexChanges) (not .ForeignKeyChanges)}}
                            <div class="diff-row modify">
                                <div class="diff-indicator">i</div>
                                <div class="diff-content">{{.Description}}</div>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                    {{if .HasViewChanges}}
                    <div class="diff-section">
                        <div class="diff-header open" onclick="this.classList.toggle('open')">
                            <svg class="diff-chevron" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M6.22 3.22a.75.75 0 011.06 0l4.25 4.25a.75.75 0 010 1.06l-4.25 4.25a.75.75 0 01-1.06-1.06L9.94 8 6.22 4.28a.75.75 0 010-1.06z"/>
                            </svg>
                            <span class="diff-table">Views</span>
                            <span class="diff-type"><span class="badge badge-warning">{{.Summary.ViewsChanged}} change(s)</span></span>
                        </div>
                        <div class="diff-body">
                            {{range .ViewChanges}}
                            <div class="diff-row {{if eq .ChangeType "added"}}add{{else if eq .ChangeType "removed"}}remove{{else}}modify{{end}}">
                                <div class="diff-indicator">{{if eq .ChangeType "added"}}+{{else if eq .ChangeType "removed"}}−{{else}}~{{end}}</div>
                                <div class="diff-content">
                                    <span class="diff-col">{{.Name}}</span>
                                    <span class="diff-detail">{{.Description}}</span>
                                </div>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                    {{if .HasRoutineChanges}}
                    <div class="diff-section">
                        <div class="diff-header open" onclick="this.classList.toggle('open')">
                            <svg class="diff-chevron" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M6.22 3.22a.75.75 0 011.06 0l4.25 4.25a.75.75 0 010 1.06l-4.25 4.25a.75.75 0 01-1.06-1.06L9.94 8 6.22 4.28a.75.75 0 010-1.06z"/>
                            </svg>
                            <span class="diff-table">Routines</span>
                            <span class="diff-type"><span class="badge badge-warning">{{.Summary.RoutinesChanged}} change(s)</span></span>
                        </div>
                        <div class="diff-body">
                            {{range .RoutineChanges}}
                            <div class="diff-row {{if eq .ChangeType "added"}}add{{else if eq .ChangeType "removed"}}remove{{else}}modify{{end}}">
                                <div class="diff-indicator">{{if eq .ChangeType "added"}}+{{else if eq .ChangeType "removed"}}−{{else}}~{{end}}</div>
                                <div class="diff-content">
                                    <span class="diff-col">{{if .Kind}}{{.Kind}}: {{end}}{{.Name}}</span>
                                    <span class="diff-detail">{{.Description}}</span>
                                </div>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                    {{if .HasTriggerChanges}}
                    <div class="diff-section">
                        <div class="diff-header open" onclick="this.classList.toggle('open')">
                            <svg class="diff-chevron" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M6.22 3.22a.75.75 0 011.06 0l4.25 4.25a.75.75 0 010 1.06l-4.25 4.25a.75.75 0 01-1.06-1.06L9.94 8 6.22 4.28a.75.75 0 010-1.06z"/>
                            </svg>
                            <span class="diff-table">Triggers</span>
                            <span class="diff-type"><span class="badge badge-warning">{{.Summary.TriggersChanged}} change(s)</span></span>
                        </div>
                        <div class="diff-body">
                            {{range .TriggerChanges}}
                            <div class="diff-row {{if eq .ChangeType "added"}}add{{else if eq .ChangeType "removed"}}remove{{else}}modify{{end}}">
                                <div class="diff-indicator">{{if eq .ChangeType "added"}}+{{else if eq .ChangeType "removed"}}−{{else}}~{{end}}</div>
                                <div class="diff-content">
                                    <span class="diff-col">{{.Name}}{{if .Table}} (table: {{.Table}}){{end}}</span>
                                    <span class="diff-detail">{{.Description}}</span>
                                </div>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                    {{if .HasSequenceChanges}}
                    <div class="diff-section">
                        <div class="diff-header open" onclick="this.classList.toggle('open')">
                            <svg class="diff-chevron" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M6.22 3.22a.75.75 0 011.06 0l4.25 4.25a.75.75 0 010 1.06l-4.25 4.25a.75.75 0 01-1.06-1.06L9.94 8 6.22 4.28a.75.75 0 010-1.06z"/>
                            </svg>
                            <span class="diff-table">Sequences</span>
                            <span class="diff-type"><span class="badge badge-warning">{{.Summary.SequencesChanged}} change(s)</span></span>
                        </div>
                        <div class="diff-body">
                            {{range .SequenceChanges}}
                            <div class="diff-row {{if eq .ChangeType "added"}}add{{else if eq .ChangeType "removed"}}remove{{else}}modify{{end}}">
                                <div class="diff-indicator">{{if eq .ChangeType "added"}}+{{else if eq .ChangeType "removed"}}−{{else}}~{{end}}</div>
                                <div class="diff-content">
                                    <span class="diff-col">{{.Name}}</span>
                                    <span class="diff-detail">{{.Description}}</span>
                                </div>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                {{else}}
                    <div class="empty">
                        <div class="empty-title">No schema changes</div>
                        <div class="empty-desc">Source and target schemas are identical</div>
                    </div>
                {{end}}
            </div>

            <div id="tab-data" class="tab-panel" data-title="Data Changes">
                {{if .HasDataDiff}}
                    <div class="filters">
                        <select class="filter-select" id="table-filter" onchange="filterTable()">
                            <option value="">All tables</option>
                            {{range .TableDiffs}}{{if .HasChanges}}<option value="{{.Table}}">{{.Table}}</option>{{end}}{{end}}
                        </select>
                    </div>
                    <div class="table-wrap">
                        <table id="data-table">
                            <thead>
                                <tr>
                                    <th>Table</th>
                                    <th>Added</th>
                                    <th>Removed</th>
                                    <th>Modified</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range .TableDiffs}}{{if .HasChanges}}
                                <tr data-table="{{.Table}}" class="data-row" onclick="toggleKeys(this)">
                                    <td>
                                        <span class="mono">{{.Table}}</span>
                                        {{if or .AddedKeys .RemovedKeys .UpdatedKeys}}<span class="expand-hint">(click to expand)</span>{{end}}
                                    </td>
                                    <td>{{if gt .AddedCount 0}}<span class="change change-add">+{{.AddedCount}}</span>{{else}}<span class="change">—</span>{{end}}</td>
                                    <td>{{if gt .RemovedCount 0}}<span class="change change-remove">−{{.RemovedCount}}</span>{{else}}<span class="change">—</span>{{end}}</td>
                                    <td>{{if gt .UpdatedCount 0}}<span class="change change-modify">~{{.UpdatedCount}}</span>{{else}}<span class="change">—</span>{{end}}</td>
                                </tr>
                                {{if or .AddedKeys .RemovedKeys .UpdatedKeys}}
                                <tr class="keys-row" data-table="{{.Table}}" style="display:none;">
                                    <td colspan="4">
                                        <div class="keys-detail">
                                            {{if .AddedKeys}}
                                            <div class="keys-section">
                                                <span class="keys-label change-add">Added Keys:</span>
                                                <div class="keys-list">{{range .AddedKeys}}<span class="key-item">{{.}}</span>{{end}}</div>
                                            </div>
                                            {{end}}
                                            {{if .RemovedKeys}}
                                            <div class="keys-section">
                                                <span class="keys-label change-remove">Removed Keys:</span>
                                                <div class="keys-list">{{range .RemovedKeys}}<span class="key-item">{{.}}</span>{{end}}</div>
                                            </div>
                                            {{end}}
                                            {{if .UpdatedKeys}}
                                            <div class="keys-section">
                                                <span class="keys-label change-modify">Modified Keys:</span>
                                                <div class="keys-list">{{range .UpdatedKeys}}<span class="key-item">{{.}}</span>{{end}}</div>
                                            </div>
                                            {{end}}
                                        </div>
                                    </td>
                                </tr>
                                {{end}}
                                {{end}}{{end}}
                            </tbody>
                        </table>
                    </div>
                {{else}}
                    <div class="empty">
                        <div class="empty-title">No data changes</div>
                        <div class="empty-desc">Source and target data are identical</div>
                    </div>
                {{end}}
            </div>

            <div id="tab-conflicts" class="tab-panel" data-title="Conflicts">
                {{if .HasConflicts}}
                    <div class="filters">
                        <select class="filter-select" id="conflict-filter" onchange="filterConflicts()">
                            <option value="">All tables</option>
                        </select>
                    </div>
                    <div class="conflict-list" id="conflict-list">
                        {{range .ConflictItems}}
                        <div class="conflict-item" data-table="{{.Table}}">
                            <span class="conflict-table">{{.Table}}</span>
                            <span class="conflict-key">{{.Key}}</span>
                            <div class="conflict-hashes">
                                <span class="hash">{{.ProdHash}}</span>
                                <span class="hash-arrow">→</span>
                                <span class="hash">{{.DevHash}}</span>
                            </div>
                            <div class="conflict-badges">
                                {{if .Strategy}}
                                <span class="badge badge-strategy {{if eq .Strategy "ours"}}badge-neutral{{else if eq .Strategy "theirs"}}badge-success{{else}}badge-warning{{end}}">{{.Strategy}}</span>
                                {{end}}
                                {{if .IsResolved}}
                                    {{if eq .Decision "keep_prod"}}<span class="badge badge-neutral">Keep Source</span>
                                    {{else if eq .Decision "use_dev"}}<span class="badge badge-success">Use Target</span>
                                    {{else}}<span class="badge badge-warning">Pending</span>{{end}}
                                {{else}}<span class="badge badge-warning">Pending</span>{{end}}
                            </div>
                        </div>
                        {{end}}
                    </div>
                {{else}}
                    <div class="empty">
                        <div class="empty-title">No conflicts</div>
                        <div class="empty-desc">No conflicting rows between source and target</div>
                    </div>
                {{end}}
            </div>

            {{if .HasMigration}}
            <div id="tab-sql" class="tab-panel" data-title="Migration SQL">
                <div class="sql-wrap">
                    <div class="sql-header">
                        <span class="sql-title">{{if .MigrationPack}}{{.MigrationPack}}{{else}}migration.sql{{end}}</span>
                        <button class="btn" onclick="copySQL()">Copy</button>
                    </div>
                    <div class="sql-code">
                        <pre id="sql-content">{{.MigrationSQL}}</pre>
                    </div>
                </div>
            </div>
            {{end}}
        </div>
    </div>

    <script>
        function switchTab(id) {
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
            document.querySelector('[data-tab="'+id+'"]').classList.add('active');
            document.getElementById('tab-'+id).classList.add('active');
        }

        function filterTable() {
            const v = document.getElementById('table-filter').value;
            document.querySelectorAll('#data-table tbody tr.data-row').forEach(r => {
                const show = !v || r.dataset.table === v;
                r.style.display = show ? '' : 'none';
                const keysRow = r.nextElementSibling;
                if (keysRow && keysRow.classList.contains('keys-row')) {
                    keysRow.style.display = 'none';
                }
            });
        }

        function toggleKeys(row) {
            const keysRow = row.nextElementSibling;
            if (keysRow && keysRow.classList.contains('keys-row')) {
                keysRow.style.display = keysRow.style.display === 'none' ? '' : 'none';
            }
        }

        function filterConflicts() {
            const v = document.getElementById('conflict-filter').value;
            document.querySelectorAll('#conflict-list .conflict-item').forEach(r => {
                r.style.display = !v || r.dataset.table === v ? '' : 'none';
            });
        }

        function copySQL() {
            const t = document.getElementById('sql-content').textContent;
            navigator.clipboard.writeText(t).then(() => {
                const b = event.target;
                b.textContent = 'Copied';
                setTimeout(() => b.textContent = 'Copy', 1500);
            });
        }

        function highlightSQL(code) {
            const kw = ['SELECT','INSERT','UPDATE','DELETE','FROM','WHERE','INTO','VALUES','SET','CREATE','ALTER','DROP','TABLE','COLUMN','INDEX','ADD','PRIMARY','KEY','FOREIGN','REFERENCES','NOT','NULL','DEFAULT','BEGIN','COMMIT','ROLLBACK','AND','OR','IN','ON','CASCADE','UNIQUE','CONSTRAINT','MODIFY','IF','EXISTS','CASE','WHEN','THEN','ELSE','END'];
            let h = code.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
            h = h.replace(/(--.*$)/gm,'<span class="sql-comment">$1</span>');
            h = h.replace(/('(?:[^'\\]|\\.)*')/g,'<span class="sql-string">$1</span>');
            h = h.replace(/\b(\d+)\b/g,'<span class="sql-number">$1</span>');
            h = h.replace(new RegExp('\\b('+kw.join('|')+')\\b','gi'),'<span class="sql-keyword">$1</span>');
            return h;
        }

        document.addEventListener('DOMContentLoaded', function() {
            const sql = document.getElementById('sql-content');
            if (sql) sql.innerHTML = highlightSQL(sql.textContent);

            const cf = document.getElementById('conflict-filter');
            if (cf) {
                const ts = new Set();
                document.querySelectorAll('#conflict-list .conflict-item').forEach(i => ts.add(i.dataset.table));
                ts.forEach(t => { const o = document.createElement('option'); o.value = t; o.textContent = t; cf.appendChild(o); });
            }
        });
    </script>
</body>
</html>
`
