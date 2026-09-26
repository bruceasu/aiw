# CLI 涓庢墿灞曞叆鍙?

## Purpose

鍥哄畾 AIW 鍘熺敓鍛戒护銆佸垵濮嬪寲銆佹彃浠舵墿灞曞強鐙珛闂瓟/鎻愪氦杈呭姪鍏ュ彛鐨勫叕鍏辫涓恒€傛彃浠跺悇鑷殑瀹屾暣涓氬姟璇箟涓嶅湪鏈熀绾挎灇涓捐寖鍥村唴銆?

## Requirements

### Requirement: 鍐呯疆涓庢彃浠跺垎娲?

涓诲叆鍙?MUST 浼樺厛澶勭悊鍐呯疆鍛戒护锛汿ask 鍛戒护鏀寔椤跺眰鍏ュ彛鍙?`task <command>` 璺緞銆傛湭鐭ラ《灞傚悕绉?MUST 灏濊瘯鍙戠幇 `aiw-<name>` 鎻掍欢銆倃t MUST 缁忔彃浠跺叆鍙ｆ墽琛屻€?

#### Scenario: 璋冪敤宸ヤ綔娴佸懡浠?

- **WHEN** 使用 `aiw wf ...`
- **THEN** 命令进入 Workflow facade，并统一分派到 Task workflow。

#### Scenario: 璋冪敤澶栭儴鎵╁睍

- **WHEN** 椤跺眰鍛戒护娌℃湁鍐呯疆澶勭悊鍣ㄤ笖鍙戠幇浜嗗搴旀彃浠?
- **THEN** 绯荤粺鎶婂墿浣欏弬鏁颁氦缁欐彃浠舵墽琛岋紝骞朵紶鍏ユ彃浠跺悕绉般€佽矾寰勫拰 AIW 璋冪敤鐜銆?

#### Scenario: 旧工作流入口已移除

- **WHEN** 使用 `aiw workflow ...` 或 `aiw task workflow ...`
- **THEN** AIW 拒绝调用并提示使用 `aiw wf ...`。

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

## 瀹炵幇渚濇嵁

- [涓诲叆鍙(../../../main.go)銆乕Task 鍒嗘淳](../../../internal/commands/task/command.go)銆?
- [鎻掍欢鍙戠幇](../../../internal/plugin/discover.go)銆乕鎻掍欢鎵ц](../../../internal/plugin/exec.go)銆乕鍒濆鍖朷(../../../internal/commands/task/init.go)銆?
- [ask 閰嶇疆](../../../internal/commands/ask/config.go)銆乕璺緞绛栫暐](../../../internal/commands/ask/path_policy.go)銆乕闂瓟鎵ц](../../../internal/commands/ask/command.go)銆?
- [cz](../../../internal/cz/command.go)銆乕Git export](../../../plugins/aiw-git/git-export.py)銆?
- [ask 鍥炲綊鏉愭枡](../../../internal/commands/ask/command_test.go)锛涙湰娆℃湭鎵ц銆?

%% ReadOnly 鏄紶缁?provider 鐨勬墽琛岄厤缃紱澶栭儴 CLI 鐨勫疄闄呮枃浠惰闂殧绂讳粛鍙栧喅浜庡叾瀹炵幇锛宎llow-path 涓嶈兘琚硾鍖栦负鎵€鏈夊悗绔殑涓ユ牸璇诲彇鐧藉悕鍗曘€?
