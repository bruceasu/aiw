# Notes
Temporary findings, debugging notes, experiments.

我希望增加一个plugin 的扩展功能。
1. 类似 git 的子命令扩展方式。 即子命令可以是一个独立的命令，在builtin 命令中没有时，自动搜索对命令。
2. 搜索空间为，程序目录的子目录plugins, $HOME/.config/aiw/plugins, $PATH
3. 规则：
    - plugins 目录下
      1. 如果是文件，则文件的基础文件名为 aiw-<plugin-name>， 扩展名为 .exe, .py, .sh, .bat, .cmd, .ps1, .js 或者为空（liunx elf 格式或者linux下的脚本，第1行为#！<解析器>） 
      2. 如果目录，在继续下一层目录找 文件的基础文件名为 aiw-<plugin-name>， 扩展名为 .exe, .py, .sh, .bat, .cmd, .ps1, .js 或者为空（liunx elf 格式或者linux下的脚本，第1行为#！<解析器>） 
    - $PATH 则只考虑文件，文件的基础文件名为 aiw-<plugin-name>， 扩展名为 .exe, .py, .sh, .bat, .cmd, .ps1, .js 或者为空（liunx elf 格式或者linux下的脚本，第1行为#！<解析器>） 
4. 程序执行/脚本解析器
   - .exe / 空（linux elf）由操作系统调用 / 空（文本），有第1行决定
   - .sh 由第1行决定，默认为bash    
   - .bat/.cmd cmd.exe
   - .ps1 pwoershell, pwsh.exe
   - .js 有 bun 或者 node 执行 （建议提供用bat/sh来启动, 自行决定用 bun/node）
   - 优先级： .bat/.cmd/.sh > .py >  empty extension name script > native image
5. 用户使用子命令的方式执行plugin. aiw <plugin-name> [args...], aiw main 需要把环境变量信息，自身信息等注入到子命令的环境变量中。 