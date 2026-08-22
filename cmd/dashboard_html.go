package cmd

func getDashboardHTML(repoName string) string {
	return `<!DOCTYPE html>
<html class="dark" lang="en">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<title>kanata · ` + repoName + `</title>
<script src="https://cdn.tailwindcss.com?plugins=forms,container-queries"></script>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;700&family=Inter:wght@400;500;600&family=Material+Symbols+Outlined:wght,FILL@100..700,0..1&display=swap" rel="stylesheet"/>
<script id="tailwind-config">
tailwind.config = {
    darkMode: "class",
    theme: {
        extend: {
            colors: {
                "tertiary-fixed": "#e2e2e2",
                "tertiary": "#ffffff",
                "secondary": "#c8c6c5",
                "primary-container": "#e2e2e2",
                "secondary-fixed": "#e5e2e1",
                "surface-container-lowest": "#0e0e0e",
                "outline": "#8e9192",
                "on-secondary": "#313030",
                "surface-container": "#201f1f",
                "tertiary-fixed-dim": "#c6c6c7",
                "primary-fixed": "#e2e2e2",
                "error-container": "#93000a",
                "on-error": "#690005",
                "on-primary-fixed": "#1a1c1c",
                "error": "#ffb4ab",
                "on-tertiary-fixed": "#1a1c1c",
                "surface": "#131313",
                "surface-tint": "#c6c6c7",
                "inverse-on-surface": "#313030",
                "surface-dim": "#131313",
                "secondary-container": "#474746",
                "on-tertiary-container": "#636565",
                "outline-variant": "#444748",
                "on-primary-container": "#636565",
                "secondary-fixed-dim": "#c8c6c5",
                "on-primary": "#2f3131",
                "primary-fixed-dim": "#c6c6c7",
                "background": "#131313",
                "on-primary-fixed-variant": "#454747",
                "surface-variant": "#353534",
                "inverse-primary": "#5d5f5f",
                "surface-container-high": "#2a2a2a",
                "primary": "#ffffff",
                "on-secondary-fixed-variant": "#474746",
                "on-background": "#e5e2e1",
                "on-tertiary-fixed-variant": "#454747",
                "surface-bright": "#3a3939",
                "surface-container-highest": "#353534",
                "on-error-container": "#ffdad6",
                "tertiary-container": "#e2e2e2",
                "on-surface": "#e5e2e1",
                "inverse-surface": "#e5e2e1",
                "on-secondary-fixed": "#1c1b1b",
                "surface-container-low": "#1c1b1b",
                "on-surface-variant": "#c4c7c8",
                "on-tertiary": "#2f3131",
                "on-secondary-container": "#b7b5b4"
            },
            fontFamily: {
                "code-md": ["JetBrains Mono"],
                "ui-label-md": ["Inter"],
                "ui-label-sm": ["Inter"],
                "ui-label-lg": ["Inter"],
                "hash-display": ["JetBrains Mono"],
                "code-sm": ["JetBrains Mono"]
            },
            fontSize: {
                "code-md": ["13px", { lineHeight: "20px", fontWeight: "400" }],
                "ui-label-md": ["13px", { lineHeight: "18px", fontWeight: "400" }],
                "ui-label-sm": ["11px", { lineHeight: "14px", fontWeight: "500" }],
                "ui-label-lg": ["15px", { lineHeight: "20px", fontWeight: "600" }],
                "hash-display": ["12px", { lineHeight: "12px", letterSpacing: "-0.02em", fontWeight: "700" }],
                "code-sm": ["12px", { lineHeight: "16px", fontWeight: "400" }]
            }
        }
    }
}
</script>
<style>
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: #0e0e0e; }
::-webkit-scrollbar-thumb { background: #353534; }
::-webkit-scrollbar-thumb:hover { background: #444748; }

.semantic-addition { border-color: #22c55e !important; color: #22c55e !important; }
.semantic-modification { border-color: #eab308 !important; color: #eab308 !important; }
.semantic-deletion { border-color: #ef4444 !important; color: #ef4444 !important; }
</style>
</head>
<body class="bg-surface-container-lowest text-on-surface font-ui-label-md lowercase h-screen w-screen overflow-hidden flex flex-col m-0 p-0">

<!-- TopAppBar -->
<header class="bg-surface-container-lowest full-width top-0 border-b border-outline-variant flex items-center justify-between w-full h-10 px-4 transition-none flex-shrink-0">
  <div class="flex items-center space-x-4">
    <span class="font-hash-display text-hash-display font-bold text-primary">kanata</span>
    <div class="flex items-center space-x-3 border-l border-outline-variant pl-4 h-4">
      <span class="material-symbols-outlined text-[16px] text-on-surface-variant">search</span>
      <input id="searchInput" oninput="filterSnapshots(this.value)" class="bg-transparent border-none p-0 h-full text-ui-label-sm font-ui-label-sm text-on-surface focus:ring-0 placeholder:text-on-surface-variant w-48" placeholder="search snapshots..." type="text"/>
    </div>
  </div>

  <nav class="flex items-center space-x-6 font-ui-label-sm text-ui-label-sm lowercase h-full">
    <button onclick="setMainView('diff')" id="nav-diff" class="text-primary border-b-2 border-primary pb-1 h-full flex items-center">semantic_diff</button>
    <button onclick="setMainView('tree')" id="nav-tree" class="text-on-surface-variant hover:text-on-surface h-full flex items-center px-2">ast_explorer</button>
  </nav>

  <div class="flex items-center space-x-3 text-on-surface-variant">
    <button onclick="refreshData()" title="Refresh" class="hover:text-on-surface flex items-center justify-center w-6 h-6"><span class="material-symbols-outlined text-[16px]">sync</span></button>
  </div>
</header>

<!-- Main Content Grid -->
<main class="flex flex-1 overflow-hidden h-[calc(100vh-40px)]">

  <!-- Left Sidebar (Streams) -->
  <aside class="bg-surface-container-lowest text-primary font-ui-label-md text-ui-label-md lowercase h-full w-48 border-r border-outline-variant flex flex-col overflow-y-auto flex-shrink-0">
    <div class="p-2 text-on-surface-variant font-ui-label-sm text-ui-label-sm uppercase tracking-wider mb-1 mt-2">streams</div>
    <div id="streamsList" class="flex flex-col space-y-[1px]">
      <!-- Streams rendered dynamically -->
    </div>
  </aside>

  <!-- Middle Column (Snapshot Timeline) -->
  <section class="w-80 border-r border-outline-variant bg-surface flex flex-col flex-shrink-0 h-full overflow-hidden">
    <div class="p-2 border-b border-outline-variant bg-surface-container-lowest flex items-center justify-between flex-shrink-0 h-10">
      <span class="font-ui-label-sm text-ui-label-sm text-on-surface-variant uppercase tracking-wider">timeline</span>
      <span id="snapshotCount" class="font-code-sm text-code-sm text-on-surface-variant">0 snapshots</span>
    </div>
    
    <div id="snapshotList" class="flex-1 overflow-y-auto flex flex-col">
      <!-- Snapshot cards rendered dynamically -->
    </div>
  </section>

  <!-- Right Main Panel (Semantic AST Diff & Explorer) -->
  <section id="panel-diff" class="flex-1 bg-surface-dim flex flex-col overflow-hidden h-full">
    <header class="h-10 border-b border-outline-variant bg-surface-container-lowest px-4 flex items-center justify-between flex-shrink-0">
      <div class="flex items-center gap-2">
        <span class="material-symbols-outlined text-[16px] text-on-surface-variant">compare_arrows</span>
        <span id="diffComparingLabel" class="font-code-sm text-code-sm text-on-surface">comparing: head vs workspace</span>
      </div>
      <div id="diffStats" class="flex gap-2">
        <span class="px-2 py-0.5 border border-outline-variant font-code-sm text-code-sm text-on-surface-variant bg-surface-container-lowest flex items-center gap-1"><span class="w-2 h-2 semantic-addition border bg-transparent inline-block"></span> <span id="statAdd">0</span> add</span>
        <span class="px-2 py-0.5 border border-outline-variant font-code-sm text-code-sm text-on-surface-variant bg-surface-container-lowest flex items-center gap-1"><span class="w-2 h-2 semantic-modification border bg-transparent inline-block"></span> <span id="statMod">0</span> mod</span>
        <span class="px-2 py-0.5 border border-outline-variant font-code-sm text-code-sm text-on-surface-variant bg-surface-container-lowest flex items-center gap-1"><span class="w-2 h-2 semantic-deletion border bg-transparent inline-block"></span> <span id="statDel">0</span> del</span>
      </div>
    </header>

    <div id="diffContainer" class="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
      <!-- Dynamic Semantic File & Node Diff Groups -->
    </div>
  </section>

  <!-- Alternative Right Panel: AST Tree Explorer -->
  <section id="panel-tree" class="hidden flex-1 bg-surface-dim flex flex-col overflow-hidden h-full">
    <header class="h-10 border-b border-outline-variant bg-surface-container-lowest px-4 flex items-center justify-between flex-shrink-0">
      <div class="flex items-center gap-2">
        <span class="material-symbols-outlined text-[16px] text-on-surface-variant">account_tree</span>
        <span class="font-code-sm text-code-sm text-on-surface">ast architecture explorer</span>
      </div>
      <span id="treeStats" class="font-code-sm text-code-sm text-on-surface-variant">0 files</span>
    </header>
    <div class="grid grid-cols-12 flex-1 overflow-hidden">
      <div id="treeFileList" class="col-span-4 border-r border-outline-variant overflow-y-auto p-2 flex flex-col gap-1"></div>
      <div id="treeNodeList" class="col-span-8 overflow-y-auto p-4 flex flex-col gap-3"></div>
    </div>
  </section>

</main>

<script>
var currentStream = 'main';
var allSnapshots = [];
var selectedSnapshotHash = '';
var activeTree = {};
var selectedFile = '';

async function init() {
  await loadStreams();
  await refreshData();
}

async function loadStreams() {
  var res = await fetch('/api/streams');
  var streams = await res.json();
  var container = document.getElementById('streamsList');
  container.innerHTML = '';

  streams.forEach(function(s) {
    var isActive = (s.name === currentStream);
    var div = document.createElement('div');
    div.className = (isActive 
      ? 'border-l-2 border-primary bg-surface-container text-primary font-bold cursor-pointer flex items-center px-2 py-1 h-8' 
      : 'text-on-surface-variant hover:bg-surface-container-low cursor-pointer flex items-center px-2 py-1 h-8 border-l-2 border-transparent transition-colors');
    div.onclick = function() { switchStream(s.name); };
    div.innerHTML = '<span class="material-symbols-outlined text-[16px] mr-2">' + (s.name === 'main' ? 'save_as' : 'fork_right') + '</span><span>' + s.name + '</span>';
    container.appendChild(div);
  });
}

async function switchStream(stream) {
  currentStream = stream;
  selectedSnapshotHash = '';
  await loadStreams();
  await refreshData();
}

async function refreshData() {
  // Load snapshots for stream
  var res = await fetch('/api/snapshots?stream=' + encodeURIComponent(currentStream));
  allSnapshots = await res.json();
  renderTimeline(allSnapshots);

  if (allSnapshots.length > 0 && !selectedSnapshotHash) {
    selectSnapshot(allSnapshots[0].hash);
  } else if (selectedSnapshotHash) {
    selectSnapshot(selectedSnapshotHash);
  } else {
    loadWorkspaceDiff();
  }

  // Load Tree
  var treeRes = await fetch('/api/tree');
  activeTree = await treeRes.json();
  renderTree();
}

function renderTimeline(snapshots) {
  var container = document.getElementById('snapshotList');
  document.getElementById('snapshotCount').textContent = snapshots.length + ' snapshots';
  container.innerHTML = '';

  snapshots.forEach(function(snap) {
    var isSelected = (snap.hash === selectedSnapshotHash);
    var card = document.createElement('div');
    card.className = (isSelected 
      ? 'border-l-2 border-primary bg-surface-container-high p-3 cursor-pointer border-b border-outline-variant flex flex-col gap-1 hover:bg-surface-bright'
      : 'border-l-2 border-transparent p-3 cursor-pointer border-b border-outline-variant flex flex-col gap-1 hover:bg-surface-container-low');
    card.onclick = function() { selectSnapshot(snap.hash); };

    var shortHash = snap.hash.substring(0, 8);
    var timeAgo = formatTimeAgo(new Date(snap.timestamp));

    card.innerHTML = 
      '<div class="flex justify-between items-start">' +
        '<span class="font-hash-display text-hash-display ' + (isSelected ? 'text-on-surface' : 'text-on-surface-variant') + '">' + shortHash + '</span>' +
        '<span class="font-ui-label-sm text-ui-label-sm text-on-surface-variant">' + timeAgo + '</span>' +
      '</div>' +
      '<p class="font-ui-label-md text-ui-label-md ' + (isSelected ? 'text-on-surface' : 'text-on-surface-variant') + ' leading-snug">' + snap.intent + '</p>' +
      '<div class="flex items-center gap-1 mt-1">' +
        '<span class="font-ui-label-sm text-ui-label-sm text-on-surface-variant">' + snap.author + '</span>' +
      '</div>';
    container.appendChild(card);
  });
}

function filterSnapshots(query) {
  if (!query) {
    renderTimeline(allSnapshots);
    return;
  }
  var q = query.toLowerCase();
  var filtered = allSnapshots.filter(function(s) {
    return s.hash.toLowerCase().includes(q) || s.intent.toLowerCase().includes(q) || s.author.toLowerCase().includes(q);
  });
  renderTimeline(filtered);
}

async function selectSnapshot(hash) {
  selectedSnapshotHash = hash;
  renderTimeline(allSnapshots);

  var snap = allSnapshots.find(function(s) { return s.hash === hash; });
  var parentHash = snap ? snap.parent_hash : '';

  document.getElementById('diffComparingLabel').textContent = 'comparing: ' + (parentHash ? parentHash.substring(0,8) : 'root') + ' vs ' + hash.substring(0, 8);

  var diffRes = await fetch('/api/diff?from=' + encodeURIComponent(parentHash) + '&to=' + encodeURIComponent(hash));
  var diffData = await diffRes.json();
  renderDiffView(diffData.diff);
}

async function loadWorkspaceDiff() {
  var diffRes = await fetch('/api/diff?to=workspace');
  var diffData = await diffRes.json();
  document.getElementById('diffComparingLabel').textContent = 'comparing: head vs workspace';
  renderDiffView(diffData.diff);
}

function renderDiffView(diff) {
  var container = document.getElementById('diffContainer');
  container.innerHTML = '';

  var files = Object.keys(diff.files || {});
  document.getElementById('statAdd').textContent = diff.added_nodes_count || 0;
  document.getElementById('statMod').textContent = diff.modified_nodes_count || 0;
  document.getElementById('statDel').textContent = diff.removed_nodes_count || 0;

  if (files.length === 0) {
    container.innerHTML = '<div class="p-8 text-center text-on-surface-variant font-code-sm">no semantic AST changes detected.</div>';
    return;
  }

  files.forEach(function(f) {
    var fd = diff.files[f];
    var group = document.createElement('div');
    group.className = 'flex flex-col border border-outline-variant bg-surface-container-lowest';

    var header = document.createElement('div');
    header.className = 'px-3 py-1.5 border-b border-outline-variant bg-surface-container-high flex items-center gap-2';
    header.innerHTML = '<span class="material-symbols-outlined text-[16px] text-on-surface-variant">description</span><span class="font-code-sm text-code-sm text-on-surface">' + fd.change_type + ' ' + f + '</span>';
    group.appendChild(header);

    var nodesContainer = document.createElement('div');
    nodesContainer.className = 'p-3 flex flex-col gap-2 pl-6 relative';
    nodesContainer.innerHTML = '<div class="absolute left-4 top-0 bottom-0 w-px bg-outline-variant"></div>';

    (fd.node_diffs || []).forEach(function(nd) {
      var nodeRow = document.createElement('div');
      nodeRow.className = 'flex flex-col relative mt-1';

      var badgeClass = 'semantic-modification';
      var prefix = '~';
      if (nd.change_type === 'added') { badgeClass = 'semantic-addition'; prefix = '+'; }
      if (nd.change_type === 'removed') { badgeClass = 'semantic-deletion'; prefix = '-'; }

      var tagHTML = 
        '<div class="flex items-center gap-2 relative">' +
          '<div class="absolute -left-6 w-4 h-px bg-outline-variant top-1/2"></div>' +
          '<div class="px-2 py-0.5 border ' + badgeClass + ' bg-surface-container flex items-center gap-2 w-fit">' +
            '<span class="font-code-sm text-code-sm">' + prefix + ' ' + (nd.signature || nd.node_name) + '</span>' +
          '</div>' +
        '</div>';

      var linesHTML = '';
      if (nd.old_node || nd.new_node) {
        var oldContent = nd.old_node ? nd.old_node.content : '';
        var newContent = nd.new_node ? nd.new_node.content : '';
        linesHTML = renderLCSDiffBlock(oldContent, newContent);
      }

      nodeRow.innerHTML = tagHTML + linesHTML;
      nodesContainer.appendChild(nodeRow);
    });

    group.appendChild(nodesContainer);
    container.appendChild(group);
  });
}

function renderLCSDiffBlock(oldText, newText) {
  var oldLines = oldText ? oldText.split('\n') : [];
  var newLines = newText ? newText.split('\n') : [];

  if (oldLines.length === 0 && newLines.length === 0) return '';

  var html = '<div class="ml-2 mt-1.5 border border-outline-variant bg-surface-container-lowest font-code-md text-code-md overflow-hidden">';
  
  if (oldLines.length === 0) {
    // Pure addition
    newLines.forEach(function(l, i) {
      html += '<div class="flex w-full bg-[#1c3f25] border-l-2 semantic-addition">' +
        '<div class="w-10 text-right pr-2 text-[#22c55e] border-r border-outline-variant select-none opacity-60">' + (i+1) + '</div>' +
        '<div class="px-2 whitespace-pre text-on-surface">+ ' + escapeHTML(l) + '</div></div>';
    });
  } else if (newLines.length === 0) {
    // Pure deletion
    oldLines.forEach(function(l, i) {
      html += '<div class="flex w-full bg-[#3f1c1c] border-l-2 semantic-deletion">' +
        '<div class="w-10 text-right pr-2 text-[#ef4444] border-r border-outline-variant select-none opacity-60">' + (i+1) + '</div>' +
        '<div class="px-2 whitespace-pre text-on-surface">- ' + escapeHTML(l) + '</div></div>';
    });
  } else {
    // Line diff
    var max = Math.max(oldLines.length, newLines.length);
    for (var i = 0; i < max; i++) {
      var o = oldLines[i];
      var n = newLines[i];
      if (o === n) {
        html += '<div class="flex w-full hover:bg-surface-container-low">' +
          '<div class="w-10 text-right pr-2 text-on-surface-variant border-r border-outline-variant select-none bg-surface-container">' + (i+1) + '</div>' +
          '<div class="px-2 whitespace-pre text-on-surface-variant">  ' + escapeHTML(n || '') + '</div></div>';
      } else {
        if (o !== undefined) {
          html += '<div class="flex w-full bg-[#3f1c1c] border-l-2 semantic-deletion">' +
            '<div class="w-10 text-right pr-2 text-[#ef4444] border-r border-outline-variant select-none opacity-50">' + (i+1) + '</div>' +
            '<div class="px-2 whitespace-pre text-on-surface">- ' + escapeHTML(o) + '</div></div>';
        }
        if (n !== undefined) {
          html += '<div class="flex w-full bg-[#1c3f25] border-l-2 semantic-addition">' +
            '<div class="w-10 text-right pr-2 text-[#22c55e] border-r border-outline-variant select-none opacity-50">' + (i+1) + '</div>' +
            '<div class="px-2 whitespace-pre text-on-surface">+ ' + escapeHTML(n) + '</div></div>';
        }
      }
    }
  }

  html += '</div>';
  return html;
}

function renderTree() {
  var fileContainer = document.getElementById('treeFileList');
  fileContainer.innerHTML = '';
  var files = Object.keys(activeTree);
  document.getElementById('treeStats').textContent = files.length + ' tracked files';

  files.forEach(function(f) {
    var count = Object.keys(activeTree[f].nodes || {}).length;
    var btn = document.createElement('button');
    btn.className = (selectedFile === f 
      ? 'w-full text-left px-2 py-1.5 bg-surface-container text-primary font-bold border-l-2 border-primary flex justify-between items-center'
      : 'w-full text-left px-2 py-1.5 text-on-surface-variant hover:bg-surface-container-low flex justify-between items-center border-l-2 border-transparent');
    btn.onclick = function() { selectTreeFile(f); };
    btn.innerHTML = '<span class="truncate font-code-sm">' + f + '</span><span class="font-code-sm text-xs opacity-60">' + count + '</span>';
    fileContainer.appendChild(btn);
  });

  if (files.length > 0 && (!selectedFile || !activeTree[selectedFile])) {
    selectTreeFile(files[0]);
  }
}

function selectTreeFile(file) {
  selectedFile = file;
  renderTree();
  var container = document.getElementById('treeNodeList');
  container.innerHTML = '';
  var f = activeTree[file];
  if (!f || !f.nodes) return;

  var nodeIDs = Object.keys(f.nodes);
  nodeIDs.forEach(function(id) {
    var node = f.nodes[id];
    var card = document.createElement('div');
    card.className = 'border border-outline-variant bg-surface-container-lowest p-3 flex flex-col gap-2';
    card.innerHTML = 
      '<div class="flex items-center justify-between">' +
        '<span class="font-code-sm text-primary font-bold">' + (node.signature || node.name) + '</span>' +
        '<span class="text-[10px] font-mono border border-outline-variant px-1.5 py-0.5 text-on-surface-variant">' + node.type + '</span>' +
      '</div>' +
      '<pre class="p-2 bg-surface-container font-code-md text-xs text-on-surface overflow-x-auto whitespace-pre">' + escapeHTML(node.content || '') + '</pre>';
    container.appendChild(card);
  });
}

function setMainView(tab) {
  if (tab === 'diff') {
    document.getElementById('panel-diff').classList.remove('hidden');
    document.getElementById('panel-tree').classList.add('hidden');
    document.getElementById('nav-diff').className = 'text-primary border-b-2 border-primary pb-1 h-full flex items-center';
    document.getElementById('nav-tree').className = 'text-on-surface-variant hover:text-on-surface h-full flex items-center px-2';
  } else {
    document.getElementById('panel-diff').classList.add('hidden');
    document.getElementById('panel-tree').classList.remove('hidden');
    document.getElementById('nav-tree').className = 'text-primary border-b-2 border-primary pb-1 h-full flex items-center';
    document.getElementById('nav-diff').className = 'text-on-surface-variant hover:text-on-surface h-full flex items-center px-2';
  }
}

function formatTimeAgo(date) {
  var sec = Math.floor((new Date() - date) / 1000);
  if (sec < 60) return 'just now';
  if (sec < 3600) return Math.floor(sec / 60) + ' min ago';
  if (sec < 86400) return Math.floor(sec / 3600) + ' hr ago';
  return Math.floor(sec / 86400) + ' day ago';
}

function escapeHTML(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

window.onload = init;
</script>
</body>
</html>`
}
