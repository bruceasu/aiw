# CLI plugins

## Purpose

为 AIW 提供 plugins 扩展

## Requirements

### Requirement: Built-in and plugin dispatch

AIW MUST dispatch built-in commands before external plugins. Unknown commands
MUST resolve through the `aiw-<name>` plugin convention. The removed `aiw wf`
command MUST NOT be advertised as available. Numbered FD worktrees MUST be
managed through `aiw git wt` in the `aiw-git` plugin, which directly creates
and records worktrees and provides status, commit, local-merge, and list
operations. No standalone `aiw wt` command is provided.

`aiw git wt delete <fd-id>` MUST remove only the conventional `.wt/<FD-ID>`
worktree registered on `feature/<FD-ID>` and that local branch. Worktree removal
MUST NOT force away uncommitted changes. If either resource is absent, the
command MUST report it and continue with the other resource. Workspace metadata
may be removed after cleanup succeeds; FD receipts MUST be retained.
`aiw git wt sync <fd-id> [branch]` MUST merge the primary worktree's current
local branch, or the named local branch, into the recorded FD worktree.
`aiw git wt cherry-pick <fd-id> <commit-id>` MUST cherry-pick the resolved
commit into the recorded FD worktree. Both commands MUST reject a dirty or
already conflicted FD worktree before starting and preserve Git recovery state
when an operation conflicts.

#### Scenario: Call an FD worktree command

- **WHEN** a user runs `aiw git wt add FD-001` or another supported `aiw git wt`
  operation
- **THEN** the plugin resolves the FD workspace record and performs that
  operation without creating or requiring a Task

#### Scenario: Delete an FD worktree and branch

- **WHEN** a user runs `aiw git wt delete FD-001`
- **THEN** the plugin removes the registered `.wt/FD-001` worktree and
  `feature/FD-001` branch when present, reporting an absent resource and
  continuing with the other one
- **AND** it preserves the worktree and branch if Git refuses safe removal
- **AND** it retains FD receipts

#### Scenario: Sync from a local branch

- **WHEN** a user runs `aiw git wt sync FD-001` or
  `aiw git wt sync FD-001 release`
- **THEN** the plugin merges the primary worktree's current branch or the named
  local branch into the recorded FD worktree
- **AND** it leaves merge conflicts available for resolution in that worktree

#### Scenario: Cherry-pick a commit

- **WHEN** a user runs `aiw git wt cherry-pick FD-001 <commit-id>`
- **THEN** the plugin resolves a commit object and cherry-picks it into the
  recorded FD worktree
- **AND** it leaves any conflict state available for recovery there

#### Scenario: Removed Task workflow command

- **WHEN** a user invokes the removed `aiw wf` command
- **THEN** AIW does not advertise it as a supported command

### Requirement: Guided Git patch creation and application

`aiw git patch create FILE.patch` MUST export tracked staged and unstaged changes as a binary-capable patch and a companion Markdown guide with a change summary. `--staged` and `--worktree` MUST restrict the source to that respective diff. It MUST NOT silently include untracked files or overwrite an existing patch or guide. Empty diffs MUST fail clearly.

`--from A --to B` MUST accept two commit IDs or branch refs, resolve each to a fixed commit ID, and export the direct A-to-B tree diff. Both options MUST be supplied together and MUST NOT be mixed with `--staged` or `--worktree`. The companion guide MUST record both input refs and resolved IDs. Invalid refs MUST fail before writing output.

`aiw git patch apply FILE.patch` MUST check applicability before applying to the working tree. A failed check MUST NOT run the apply step; the command MUST report the Git error and recovery advice, including an already-applied hint when a reverse check succeeds. The command MUST NOT automatically stage, commit, use three-way merge, or accept partial application.

#### Scenario: Create a patch with guidance

- **WHEN** a user exports current changes or selects staged or unstaged changes
- **THEN** AIW writes the selected binary-capable diff and a companion usage and recovery guide without overwriting existing files

#### Scenario: Apply a patch that does not match

- **WHEN** a user applies a patch and Git's applicability check fails
- **THEN** AIW leaves the repository untouched by the apply step, reports the error, and suggests how to inspect or recover

#### Scenario: Export changes between two refs

- **WHEN** a user runs `aiw git patch create transfer.patch --from A --to B` with commit IDs or branches
- **THEN** AIW writes the direct A-to-B diff and records both fixed commit IDs in the guide, without including unrelated worktree changes

### Requirement: AI-assisted Git commands

`aiw git aic` MUST stage all changes, generate a concise Conventional Commit
message using the configured CZ text provider, and pass that message to
`git commit -F -` only after successful generation. Generation or Git failure
MUST return a non-zero result; generation failure MUST NOT run `git commit`.
`aiw git air` MUST review only the staged diff and MUST NOT modify repository
state. `aiw git aib [--base REF]` MUST summarize the commit history in
`BASE..HEAD`, defaulting to `main`, and include changed file statuses and a
short change summary from `merge-base(BASE, HEAD)..HEAD`. It MUST include at
most 8,000 characters of changed-file entries and
at most 12,000 characters of diff content; omitted content MUST be marked, and
the prompt MUST tell the model not to infer omitted details. `aib` MUST NOT
modify repository state. All three commands MUST use the CZ provider
configuration and fallback behavior.

#### Scenario: Generate a commit message

- **WHEN** a user runs `aiw git aic` with changes present
- **THEN** AIW stages all changes, asks a configured CZ provider for a
  Conventional Commit message, and commits only after receiving non-empty text

#### Scenario: Review staged changes

- **WHEN** a user runs `aiw git air` with a staged diff
- **THEN** AIW prints the review response and leaves the index, worktree, and
  commit history unchanged

#### Scenario: Summarize branch history

- **WHEN** a user runs `aiw git aib` or supplies `--base REF`
- **THEN** AIW summarizes one-line commits from `main..HEAD` or `REF..HEAD`,
  includes bounded changed-file and diff context, marks omitted content, and
  does not change repository state

### Requirement: 鎻掍欢鍙戠幇涓庡瓙杩涚▼缁撴灉

鎻掍欢鍙戠幇 MUST 鎼滅储鍙墽琛屾枃浠舵梺鍜屽綋鍓嶇洰褰曠殑 plugins锛堝惈涓€绾у瓙鐩綍锛夛紝浠ュ強 PATH 涓婂尮閰嶅悕绉扮殑鍊欓€夛紝骞舵寜鎵╁睍鍚嶄紭鍏堢骇閫夊彇銆傛墽琛?MUST 浣跨敤瀵瑰簲瑙ｉ噴鍣ㄦ垨鍙墽琛屾枃浠讹紝杩炴帴鏍囧噯杈撳叆杈撳嚭锛屽苟灏嗗惎鍔ㄩ敊璇垨闈為浂閫€鍑轰綔涓哄け璐ュ弽棣堛€?

#### Scenario: 鎻掍欢鍚姩澶辫触

- **WHEN** 鎻掍欢瑙ｉ噴鍣ㄤ笉瀛樺湪鎴栧瓙杩涚▼鏃犳硶鍚姩
- **THEN** AIW 鎶ュ憡鎵ц閿欒锛岃€屼笉鏄妸璋冪敤瑙嗕负鎴愬姛銆?

### Requirement: 鍩虹鍒濆鍖栦笌鍙€夎缃?

init MUST 鍒涘缓 OpenSpec銆乀ask 杩愯鐩綍銆佸伐浣滄爲鍙婄浉鍏虫寚浠ょ洰褰曪紝骞朵粎鍦ㄧ己澶辨椂鐢熸垚鍩虹鎸囦护鏂囦欢銆傚彲閫?prompts 鍚屾鎸夋樉寮忛€夐」鎵ц锛涘畼鏂?setup 鎻掍欢涓嶅彲鐢ㄦ垨澶辫触 MUST 缁欏嚭鎻愮ず骞跺厑璁稿熀纭€鍒濆鍖栦繚鐣欍€?

#### Scenario: 宸叉湁椤圭洰鎸囦护

- **WHEN** 鎵ц鍩虹 init 涓斿熀纭€鎸囦护鏂囦欢宸茬粡瀛樺湪
- **THEN** write-if-missing 璺緞涓嶈鐩栬鏂囦欢銆?

#### Scenario: 瀹樻柟 setup 涓嶅彲鐢?

- **WHEN** 鏈烦杩?setup锛屼絾鏃犳硶鍙戠幇鎴栬繍琛屽畼鏂规彃浠?
- **THEN** 绯荤粺鎶ュ憡璇ユ儏鍐碉紝缁х画淇濈暀宸插畬鎴愮殑鍩虹鍒濆鍖栥€?

### Requirement: ask 鐨勭郴缁熸彁绀轰笌璺緞杈圭晫

ask MUST 鎸夊懡浠よ鏂囨湰銆佸懡浠よ鏂囦欢銆佷釜浜?`~/.aiw/ask/config.toml` 鐨勯『搴忚В鏋愮郴缁熸彁绀恒€傛樉寮忓懡浠よ鏂囦欢鍙洿鎺ユ巿鏉冭鍙栵紱涓汉閰嶇疆寮曠敤鐨勫閮ㄦ枃浠?MUST 閫氳繃宸ヤ綔鍖恒€佷釜浜?ask 鐩綍銆乤llow-path 鎴栦氦浜掔‘璁よ幏寰楄鍙€?

#### Scenario: 闈炰氦浜掕鍙栧閮ㄩ厤缃枃浠?

- **WHEN** 閰嶇疆寮曠敤鐨勭郴缁熸彁绀烘枃浠朵綅浜庨粯璁よ寖鍥村锛屼笖娌℃湁 allow-path
- **THEN** 闈炰氦浜掕皟鐢ㄨ繑鍥炴巿鏉冮敊璇紝涓嶉潤榛樿鍙栬鏂囦欢銆?

### Requirement: ask 鐨勭粨鏋勫寲缁撴灉

ask MUST 鍚戝叡浜?AI 灞傝姹?ReadOnly 妯″紡骞舵牎楠屽搷搴?schema_version銆乻tatus銆乧apability 鍜?safety 绛夊凡瀹炵幇瀛楁銆傛棤鏁?JSON銆佹棤鏁堢姸鎬佹垨缂哄皯蹇呴渶瀵硅薄 MUST 淇濆瓨閿欒璁板綍骞惰繑鍥炲け璐ャ€?

#### Scenario: 杩斿洖鏃犳晥鍥炵瓟

- **WHEN** 妯″瀷杈撳嚭鏃犳硶閫氳繃 ask 鍝嶅簲鏍￠獙
- **THEN** 绯荤粺淇濈暀璇婃柇锛屼笉鑳藉皢鏃犳晥鍐呭鎵撳嵃涓烘甯告垚鍔熷洖绛斻€?

### Requirement: cz 鐨勬殏瀛樺尯涓庡闃?

cz MUST 鍦ㄧ敓鎴愭彁浜よ崏绋垮墠瑕佹眰瀛樺湪 staged changes銆傚畠 MUST 鏀寔 LLM 鑽夌鎴栦氦浜掑悜瀵硷紝骞舵妸鑽夌浜ょ粰 ReviewAndCommit 娴佺▼鍚庡啀璋冪敤鎻愪氦鍑芥暟銆?

#### Scenario: 娌℃湁鏆傚瓨鏀瑰姩

- **WHEN** staged changes 涓虹┖
- **THEN** cz 杩斿洖闇€瑕佸厛 git add 鐨勯敊璇紝涓嶇敓鎴愭彁浜ゃ€?

### Requirement: Git export 瀵煎嚭鎻愪氦寮曠敤

git export MUST 璋冪敤 `git archive --format=zip` 瀵煎嚭鎸囧畾 Git ref锛屾湭鎸囧畾鏃朵娇鐢?HEAD銆傚鍑哄唴瀹?MUST 鏉ヨ嚜璇ュ紩鐢ㄧ殑鎻愪氦鏍戯紝鑰屼笉鏄湭鎻愪氦鐨勫伐浣滃尯淇敼銆?

#### Scenario: 瀵煎嚭榛樿寮曠敤

- **WHEN** 涓嶆寚瀹?ref 鎵ц export
- **THEN** 瀵煎嚭褰撳墠 HEAD 鐨勫唴瀹广€?

#### Scenario: 鎸囧畾鍏朵粬鍒嗘敮銆佹爣绛炬垨鎻愪氦

- **WHEN** 鎻愪緵鍙В鏋愮殑 Git ref
- **THEN** 瀵煎嚭璇?ref锛屾棤闇€鎶婂伐浣滃尯鍒囨崲鍒拌鍒嗘敮銆?

### Requirement: Git merge-to uses an isolated worktree

The Git plugin MUST provide `merge-to <target_branch> [source_branch]` and
`merge-to --new <new_branch> <source_branch1> [source_branch2 ...]`.
The invoking checkout, working files, and index MUST remain unchanged.
Sources MUST be validated before mutation; the command MUST NOT fetch or push.

#### Scenario: Merge into an existing target

- **WHEN** the target is an existing local branch not checked out in any worktree
- **THEN** the plugin checks it out in a unique temporary worktree outside the invoking workspace and merges the explicit source, or the current branch when omitted.
- **AND** detached HEAD without an explicit source is rejected before mutation.

#### Scenario: Create a target from several sources

- **WHEN** `--new` names a nonexistent local target and one or more valid source branches
- **THEN** the target starts at the first source commit and the remaining sources merge sequentially.
- **AND** sources may be local or already available remote-tracking branches.

#### Scenario: Preserve an existing checkout

- **WHEN** an existing target is checked out in any worktree, including the invoking workspace
- **THEN** the command rejects the operation before mutation and reports that worktree path.

#### Scenario: Success and recovery

- **WHEN** all merges succeed
- **THEN** the temporary worktree is removed without force.
- **WHEN** a merge fails or is interrupted
- **THEN** the command returns failure, retains the worktree, and reports its absolute path and recovery commands.
- **AND** earlier successful merges and any newly created target remain; no automatic reset or abort occurs.
- **WHEN** worktree cleanup fails
- **THEN** the command returns failure and reports the remaining recovery path.


## 瀹炵幇渚濇嵁

- [涓诲叆鍙(../../../src/cmd/aiw/main.go)銆乕Task 鍒嗘淳](../../../src/internal/commands/task/command.go)銆?
- [鎻掍欢鍙戠幇](../../../src/internal/plugin/discover.go)銆乕鎻掍欢鎵ц](../../../src/internal/plugin/exec.go)銆乕鍒濆鍖朷(../../../src/internal/commands/task/init.go)銆?
- [ask 閰嶇疆](../../../src/internal/commands/ask/config.go)銆乕璺緞绛栫暐](../../../src/internal/commands/ask/path_policy.go)銆乕闂瓟鎵ц](../../../src/internal/commands/ask/command.go)銆?
- [cz](../../../src/plugins/aiw-cz/aiw-cz.py)；[Git export](../../../src/plugins/aiw-git/git-export.py)。

%% ReadOnly 鏄紶缁?provider 鐨勬墽琛岄厤缃紱澶栭儴 CLI 鐨勫疄闄呮枃浠惰闂殧绂讳粛鍙栧喅浜庡叾瀹炵幇锛宎llow-path 涓嶈兘琚硾鍖栦负鎵€鏈夊悗绔殑涓ユ牸璇诲彇鐧藉悕鍗曘€?
