#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
通用 Maven Java 代码Entry启动脚本。

设计目标：
- 当前目录就是 Project Root，不向上搜索。
- 支持多模块 Maven 项目。
- 支持单模块 Maven 项目。
- 支持 Maven Wrapper，当前目录存在 mvnw/mvnw.cmd 时优先使用。
- 支持指定 JAVA_HOME，并在启动 Maven 前注入环境变量。
- 支持 JDK8 / 老 Maven，默认使用 exec-maven-plugin 1.6.0。
- 支持 -q/--quiet，减少 Maven 输出。
- 支持把 --args 参数透传给 Java main(String[] args)。

多模块执行逻辑：
    1. 在 Project Root 执行：
        mvn -pl <module> -am compile install

    2. 进入模块目录执行：
        mvn org.codehaus.mojo:exec-maven-plugin:<version>:java \
          -Dexec.mainClass=<mainClass> \
          -Dexec.classpathScope=runtime

单模块执行逻辑：
    在 Project Root 执行：
        mvn compile org.codehaus.mojo:exec-maven-plugin:<version>:java \
          -Dexec.mainClass=<mainClass> \
          -Dexec.classpathScope=runtime

示例：
    python exec-java.py \
      -m exec-java-module \
      -c exec-java.Generator

    python exec-java.py \
      -q \
      -m exec-java-module \
      -c exec-java.Generator \
      --args "--table sys_param --force"

    python exec-java.py \
      --single \
      -c com.example.tools.exec-javaerator
"""

import argparse
import os
import platform
import shlex
import shutil
import subprocess
from pathlib import Path
from typing import List, Optional


# ============================================================
# Global defaults
# ============================================================
# 这些就是启动时自动使用的全局默认值。
# 一般情况下只改这里，不需要改下面的逻辑代码。

# JDK8 / 老 Maven 项目建议默认 1.6.0。
# exec-maven-plugin 3.3.0 要求 Maven 3.6.3+，老项目容易失败。
DEFAULT_EXEC_PLUGIN_VERSION = "1.6.0"

# 默认 classpath scope。
DEFAULT_CLASSPATH_SCOPE = "runtime"

# 默认是否开启 Maven quiet 模式。
# 如果希望默认少日志，可以改成 True；也可以运行时传 -q/--quiet。
DEFAULT_QUIET = False

# 默认 JAVA_HOME。
# 例如 Windows：r"C:\Program Files\Java\jdk1.8.0_202"
# 例如 macOS："/Library/Java/JavaVirtualMachines/jdk8/Contents/Home"
# 不需要固定 JDK 时保持 None。
DEFAULT_JAVA_HOME = None

# 默认 Maven 命令。
# None 表示自动检测当前目录的 mvnw/mvnw.cmd，否则使用 PATH 中的 mvn。
# 也可以写成 r"C:\apache-maven-3.6.0\bin\mvn.cmd"
DEFAULT_MAVEN_COMMAND = None
DEFAULT_FILE_ENCODING = "UTF-8"
FILE_ENCODING = "UTF-8"

EXEC_PLUGIN_GROUP_ID = "org.codehaus.mojo"
EXEC_PLUGIN_ARTIFACT_ID = "exec-maven-plugin"


# ============================================================
# Utility functions
# ============================================================

def is_windows() -> bool:
    return platform.system().lower().startswith("windows")


def project_root() -> Path:
    """
    当前目录就是 Project Root。
    不向上搜索，避免 AI CLI 或 IDE 在非预期目录执行时误判。
    """
    return Path.cwd().resolve()


def build_exec_plugin_goal(version: str) -> str:
    version = (version or "").strip()
    if not version:
        raise SystemExit("Empty exec-maven-plugin version.")
    return f"{EXEC_PLUGIN_GROUP_ID}:{EXEC_PLUGIN_ARTIFACT_ID}:{version}:java"


def find_maven_command(root: Path, user_maven: Optional[str]) -> str:
    """
    Maven 命令选择规则：
    1. 如果用户指定 --maven，则使用用户指定的命令。
    2. 如果当前目录存在 Maven Wrapper，则优先使用：
       - Windows: mvnw.cmd
       - Unix-like: ./mvnw
    3. Windows 下优先查找 mvn.cmd / mvn.bat / mvn。
    4. 其他系统查找 mvn。
    """
    if user_maven:
        user_maven_path = Path(user_maven)

        # 用户传的是完整路径。
        if user_maven_path.exists():
            return str(user_maven_path.resolve())

        # 用户传的是命令名，例如 mvn、mvn.cmd。
        found = shutil.which(user_maven)
        if found:
            return found

        raise SystemExit(f"Maven command not found: {user_maven}")

    if is_windows():
        mvnw_cmd = root / "mvnw.cmd"
        if mvnw_cmd.exists():
            return str(mvnw_cmd.resolve())

        for name in ("mvn.cmd", "mvn.bat", "mvn"):
            found = shutil.which(name)
            if found:
                return found

        raise SystemExit(
            "Maven not found. Please install Maven, add it to PATH, "
            "or specify it with --maven C:\\path\\to\\mvn.cmd"
        )

    mvnw = root / "mvnw"
    if mvnw.exists():
        return str(mvnw.resolve())

    mvn = shutil.which("mvn")
    if mvn:
        return mvn

    raise SystemExit(
        "Maven not found. Please install Maven, add it to PATH, "
        "or specify it with --maven /path/to/mvn"
    )


def build_env(java_home: Optional[str], file_encoding: Optional[str]) -> dict:
    """
    构造子进程环境变量。

    如果指定 --java，则注入 JAVA_HOME，并把 JAVA_HOME/bin 放到 PATH 前面。

    如果指定 file_encoding，则通过 MAVEN_OPTS 注入：
        -Dfile.encoding=UTF-8

    注意：
    exec-maven-plugin 的 java goal 默认在 Maven JVM 内运行，
    所以 encoding 需要尽量在 Maven JVM 启动时设置。
    """
    env = os.environ.copy()

    if java_home:
        java_home_path = Path(java_home).expanduser().resolve()

        if not java_home_path.exists():
            raise SystemExit(f"[exec-java] JAVA_HOME does not exist: {java_home_path}")

        env["JAVA_HOME"] = str(java_home_path)

        java_bin = java_home_path / "bin"
        old_path = env.get("PATH", "")
        env["PATH"] = str(java_bin) + os.pathsep + old_path

    if file_encoding:
        encoding_opt = f"-Dfile.encoding={file_encoding}"
        old_maven_opts = env.get("MAVEN_OPTS", "").strip()

        if encoding_opt not in old_maven_opts:
            if old_maven_opts:
                env["MAVEN_OPTS"] = encoding_opt + " " + old_maven_opts
            else:
                env["MAVEN_OPTS"] = encoding_opt

    return env

def quote_for_log(part: str) -> str:
    """
    仅用于打印日志。
    Windows 下 shlex.quote 的显示风格不像 cmd，因此使用 list2cmdline 更接近实际命令。
    """
    return shlex.quote(part)


def print_command(cwd: Path, cmd: List[str]) -> None:
    """
    打印即将执行的命令，方便 AI CLI / 开发者排查。
    """
    print()
    print(f"cwd: {cwd}")
    print("run:")
    if is_windows():
        print("  " + subprocess.list2cmdline(cmd))
    else:
        print("  " + " ".join(quote_for_log(part) for part in cmd))
    print()


# def run_command(cmd: List[str], cwd: Path, env: dict, dry_run: bool = False) -> None:
#     print_command(cwd, cmd)
#
#     if dry_run:
#         return
#
#     if is_windows():
#         # Windows 下 mvn 通常是 mvn.cmd。
#         # 通过 cmd /d /s /c 执行，可以让系统按终端方式解析 PATH、PATHEXT、.cmd、.bat。
#         command_line = subprocess.list2cmdline(cmd)
#         result = subprocess.run(
#             ["cmd", "/d", "/s", "/c", command_line],
#             cwd=str(cwd),
#             env=env,
#         )
#     else:
#         result = subprocess.run(
#             cmd,
#             cwd=str(cwd),
#             env=env,
#         )
#
#     if result.returncode != 0:
#         raise SystemExit(result.returncode)


def run_command(cmd: List[str], cwd: Path, env: dict, dry_run: bool = False) -> None:
    print_command(cwd, cmd)

    if dry_run:
        return

    if is_windows():
        command_line = subprocess.list2cmdline(cmd)
        popen_cmd = ["cmd", "/d", "/s", "/c", command_line]
    else:
        popen_cmd = cmd

    process = subprocess.Popen(
        popen_cmd,
        cwd=str(cwd),
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        encoding="utf-8",
        errors="replace",
        bufsize=1,
    )

    assert process.stdout is not None

    for line in process.stdout:
        print(line, end="", flush=True)

    returncode = process.wait()

    if returncode != 0:
        raise SystemExit(returncode)
        
def parse_exec_args(args_text: Optional[str]) -> List[str]:
    """
    把 --args 字符串拆成参数列表。

    例如：
        --args "--table sys_param --target module-a"

    会变成：
        ["--table", "sys_param", "--target", "module-a"]

    注意：
    - 如果参数中有空格，需要在 --args 内部再加引号。
      例如：
        --args "--name 'hello world'"
    """
    if not args_text:
        return []

    try:
        # Windows 下也使用 POSIX 风格解析，是为了让 --args 内部引号行为稳定。
        return shlex.split(args_text, posix=True)
    except ValueError as exc:
        raise SystemExit(f"Invalid --args value: {exc}") from exc


def build_exec_args_property(generator_args: List[str]) -> Optional[str]:
    """
    构造 -Dexec.args 的值。

    这里返回的是一个单独的 Maven 参数：
        -Dexec.args=--table sys_param --force

    subprocess 会把它作为一个完整参数传给 Maven。
    Windows 下再由 list2cmdline/cmd /c 正确转义。
    """
    if not generator_args:
        return None

    if is_windows():
        # Windows 命令行参数转义。
        return "-Dexec.args=" + subprocess.list2cmdline(generator_args)

    # Unix-like shell 展示/传递更适合 shlex.quote。
    return "-Dexec.args=" + " ".join(shlex.quote(arg) for arg in generator_args)


# ============================================================
# Validation
# ============================================================

def validate_main_class(main_class: str) -> None:
    """
    简单校验 Java 主类名。
    不做过度限制，允许下划线、内部类等，但不能是空字符串或包含空格。
    """
    if not main_class or not main_class.strip():
        raise SystemExit("Missing generator main class. Use -c/--class.")

    if " " in main_class.strip():
        raise SystemExit(f"Invalid main class: {main_class}")


def validate_project(root: Path) -> None:
    pom = root / "pom.xml"

    if not pom.exists():
        raise SystemExit(f"pom.xml not found in project root: {root}")


def validate_module(root: Path, module: str) -> Path:
    if not module or not module.strip():
        raise SystemExit("Missing module name. Use -m/--module or --single.")

    module_dir = root / module

    if not module_dir.exists():
        raise SystemExit(f"Module directory does not exist: {module_dir}")

    if not module_dir.is_dir():
        raise SystemExit(f"Module path is not a directory: {module_dir}")

    if not (module_dir / "pom.xml").exists():
        raise SystemExit(f"Module pom.xml not found: {module_dir / 'pom.xml'}")

    return module_dir.resolve()


# ============================================================
# Maven commands
# ============================================================

def append_maven_common_flags(cmd: List[str], quiet: bool) -> None:
    if quiet:
        cmd.append("-q")


def compile_multi_module(
    root: Path,
    module: str,
    maven: str,
    env: dict,
    dry_run: bool,
    quiet: bool = False,
) -> None:
    """
    多模块项目编译：
        mvn -q -pl <module> -am compile

    这里必须在 Project Root 下执行。
    """
    cmd = [maven]
    append_maven_common_flags(cmd, quiet)
    cmd.extend([
        "-pl",
        module,
        "-am",
        "-DskipTests",
        "-B",
        f"-Dfile.ecnoding={FILE_ENCODING}", 
        "compile",
        "install"  # 会有依赖，所以要install
    ])

    run_command(cmd, cwd=root, env=env, dry_run=dry_run)


def run_multi_module(
    module_dir: Path,
    maven: str,
    exec_plugin_goal: str,
    main_class: str,
    classpath_scope: str,
    generator_args: List[str],
    env: dict,
    dry_run: bool,
    quiet: bool = False,
) -> None:
    """
    多模块项目执行Entry：
        cd <module>
        mvn -q org.codehaus.mojo:exec-maven-plugin:<version>:java \
            -Dexec.mainClass=<mainClass> \
            -Dexec.classpathScope=runtime \
            -Dexec.args=...

    注意：
    - 这里 cwd 是 module 目录。
    - 这解决了必须在模块目录下运行的问题。
    """
    cmd = [maven]
    append_maven_common_flags(cmd, quiet)
    cmd.extend([
        f"-Dfile.ecnoding={FILE_ENCODING}", 
        exec_plugin_goal,
        "-B",
        f"-Dexec.mainClass={main_class}",
        f"-Dexec.classpathScope={classpath_scope}",
    ])

    exec_args = build_exec_args_property(generator_args)
    if exec_args:
        cmd.append(exec_args)

    run_command(cmd, cwd=module_dir, env=env, dry_run=dry_run)


def run_single_module(
    root: Path,
    maven: str,
    exec_plugin_goal: str,
    main_class: str,
    classpath_scope: str,
    generator_args: List[str],
    env: dict,
    dry_run: bool,
    quiet: bool = False,
) -> None:
    """
    单模块项目：
        mvn -q compile org.codehaus.mojo:exec-maven-plugin:<version>:java \
            -Dexec.mainClass=<mainClass> \
            -Dexec.classpathScope=runtime \
            -Dexec.args=...
    """
    cmd = [maven]
    append_maven_common_flags(cmd, quiet)
    cmd.extend([
        "compile",
        f"-Dfile.ecnoding={FILE_ENCODING}", 
        "-DskipTests", 
        "-B",
        exec_plugin_goal,
        f"-Dexec.mainClass={main_class}",
        f"-Dexec.classpathScope={classpath_scope}",
    ])

    exec_args = build_exec_args_property(generator_args)
    if exec_args:
        cmd.append(exec_args)

    run_command(cmd, cwd=root, env=env, dry_run=dry_run)


# ============================================================
# Main
# ============================================================

def main() -> None:
    parser = argparse.ArgumentParser(
        prog="exec-java.py",
        description="Run a Java Maven code generator main class in single-module or multi-module projects.",
    )

    parser.add_argument(
        "-c",
        "--class",
        dest="main_class",
        required=True,
        help="Entry主类，必须实现 main(String[] args)。",
    )

    parser.add_argument(
        "-m",
        "--module",
        dest="module",
        help="多模块项目中的模块名。单模块项目不要指定该参数。",
    )

    parser.add_argument(
        "-s",
        "--single",
        action="store_true",
        help="声明当前项目是单模块项目，不需要指定模块名。",
    )

    parser.add_argument(
        "--java",
        dest="java_home",
        default=DEFAULT_JAVA_HOME,
        help="指定 JAVA_HOME。执行 Maven 前会注入该环境变量。默认使用脚本顶部 DEFAULT_JAVA_HOME。",
    )

    parser.add_argument(
        "--maven",
        dest="maven",
        default=DEFAULT_MAVEN_COMMAND,
        help="指定 Maven 命令路径。默认自动检测 mvnw/mvnw.cmd/mvn。",
    )

    parser.add_argument(
        "--args",
        dest="generator_args",
        help='传递给 Java main 方法的参数，必须用引号括起来，例如 --args "--table sys_param --force"。',
    )

    parser.add_argument(
        "--classpath-scope",
        default=DEFAULT_CLASSPATH_SCOPE,
        choices=["compile", "runtime", "test"],
        help=f"exec-maven-plugin 的 classpathScope，默认 {DEFAULT_CLASSPATH_SCOPE}。",
    )

    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="只打印命令，不实际执行。",
    )

    parser.add_argument(
        "-q",
        "--quiet",
        action="store_true",
        default=DEFAULT_QUIET,
        help="减少 Maven 输出日志。会给 Maven 命令追加 -q。也可修改脚本顶部 DEFAULT_QUIET。",
    )

    parser.add_argument(
        "--exec-plugin-version",
        default=DEFAULT_EXEC_PLUGIN_VERSION,
        help=f"exec-maven-plugin 版本。JDK8/旧 Maven 项目建议使用 1.6.0，默认 {DEFAULT_EXEC_PLUGIN_VERSION}。 新版可以使用3.3.0",
    )

    parser.add_argument(
        "--encoding",
        default=DEFAULT_FILE_ENCODING,
        help=f"设置 Maven JVM 的 file.encoding，默认 {DEFAULT_FILE_ENCODING}。如果不想设置，可传空字符串。",
    )

    parsed = parser.parse_args()

    root = project_root()
    validate_project(root)
    validate_main_class(parsed.main_class)

    if parsed.single and parsed.module:
        raise SystemExit("--single and --module cannot be used together.")

    if not parsed.single and not parsed.module:
        raise SystemExit("Multi-module mode requires -m/--module. For single-module project, use --single.")

    FILE_ENCODING=parsed.encoding
    env = build_env(parsed.java_home, parsed.encoding)
    maven = find_maven_command(root, parsed.maven)
    exec_plugin_goal = build_exec_plugin_goal(parsed.exec_plugin_version)
    generator_args = parse_exec_args(parsed.generator_args)

    print(f"project root: {root}")
    print(f"maven: {maven}")
    print(f"exec plugin: {exec_plugin_goal}")
    print(f"classpath scope: {parsed.classpath_scope}")
    print(f"quiet: {parsed.quiet}")
    print(f"main class: {parsed.main_class}")

    if parsed.java_home:
        print(f"JAVA_HOME: {env.get('JAVA_HOME')}")

    if generator_args:
        print(f"generator args: {generator_args}")

    if parsed.single:
        print("mode: single-module")
        run_single_module(
            root=root,
            maven=maven,
            exec_plugin_goal=exec_plugin_goal,
            main_class=parsed.main_class,
            classpath_scope=parsed.classpath_scope,
            generator_args=generator_args,
            env=env,
            dry_run=parsed.dry_run,
            quiet=parsed.quiet,
        )
    else:
        print("mode: multi-module")
        print(f"module: {parsed.module}")

        module_dir = validate_module(root, parsed.module)

        compile_multi_module(
            root=root,
            module=parsed.module,
            maven=maven,
            env=env,
            dry_run=parsed.dry_run,
            quiet=parsed.quiet,
        )

        run_multi_module(
            module_dir=module_dir,
            maven=maven,
            exec_plugin_goal=exec_plugin_goal,
            main_class=parsed.main_class,
            classpath_scope=parsed.classpath_scope,
            generator_args=generator_args,
            env=env,
            dry_run=parsed.dry_run,
            quiet=parsed.quiet,
        )

    print()
    print("done.")


if __name__ == "__main__":
    main()
