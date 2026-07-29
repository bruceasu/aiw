import json
import os
import shutil
import subprocess
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory


PLUGIN = Path(__file__).resolve().parents[1] / "aiw-skills.py"
REPOSITORY_ROOT = PLUGIN.parents[2]
INSTALL_PLUGIN_SCRIPT = REPOSITORY_ROOT / "install-plugin.bat"
BUILD_SCRIPT = REPOSITORY_ROOT / "build.bat"


def write_skill(root: Path, name: str, description: str) -> Path:
    skill_dir = root / name
    skill_dir.mkdir(parents=True)
    (skill_dir / "SKILL.md").write_bytes(
        "---\n"
        "name: {}\n"
        'description: "{}"\n'
        "---\n\n"
        "Follow this workflow.\n".format(name, description).encode("utf-8"),
    )
    return skill_dir


def make_workspace(temp_dir: str) -> tuple[Path, Path, Path]:
    base = Path(temp_dir)
    source = base / "source"
    project = base / "project"
    source.mkdir()
    project.mkdir()
    return base, source, project


def make_windows_release_fixture(base: Path) -> tuple[Path, Path, dict[str, str]]:
    source_root = base / "release-source"
    plugin_dir = source_root / "plugins" / "aiw-skills"
    plugin_dir.mkdir(parents=True)
    shutil.copy2(str(PLUGIN), str(plugin_dir / "aiw-skills.py"))
    skills_root = source_root / "skills"
    skills_root.mkdir()
    write_skill(skills_root, "packaged-skill", "Installed by release script.")
    (source_root / "docs" / "usage").mkdir(parents=True)
    (source_root / "docs" / "agent-templates").mkdir(parents=True)
    binary_dir = source_root / "bin"
    binary_dir.mkdir()
    (binary_dir / "aiw-windows-amd64.exe").write_bytes(b"test binary")

    install_root = base / "installed"
    script = source_root / "install-plugin.bat"
    script.write_text(
        INSTALL_PLUGIN_SCRIPT.read_text(encoding="utf-8").replace(
            "SET INSTALL_DIR=c:\\green\\aiw",
            'SET "INSTALL_DIR={}"'.format(install_root),
        ),
        encoding="utf-8",
    )

    tools_dir = source_root
    copy_helper = tools_dir / "copy_helper.py"
    copy_helper.write_text(
        "import shutil, sys\n"
        "from pathlib import Path\n"
        "source, destination = map(Path, sys.argv[1:3])\n"
        "shutil.copytree(str(source), str(destination), dirs_exist_ok=True)\n",
        encoding="utf-8",
    )
    (tools_dir / "cp-mirror.bat").write_text(
        "@echo off\n"
        'python "%COPY_HELPER%" "%~1" "%~2"\n'
        "if errorlevel 1 exit /b 7\n"
        "exit /b 0\n",
        encoding="utf-8",
    )
    where_command = shutil.which("where.exe")
    if where_command is not None:
        shutil.copy2(where_command, str(tools_dir / "gbuild.exe"))
    env = os.environ.copy()
    env["COPY_HELPER"] = str(copy_helper)
    env["PATH"] = str(tools_dir) + os.pathsep + env["PATH"]
    return script, install_root, env


def make_skill_copy_fail(env: dict[str, str]) -> None:
    tools_dir = Path(env["COPY_HELPER"]).parent
    env["COPY_LOG"] = str(tools_dir / "copy.log")
    (tools_dir / "cp-mirror.bat").write_text(
        "@echo off\n"
        'echo %~1>>"%COPY_LOG%"\n'
        'if /I "%~1"=="skills" exit /b 7\n'
        'python "%COPY_HELPER%" "%~1" "%~2"\n'
        "if errorlevel 1 exit /b 7\n"
        "exit /b 0\n",
        encoding="utf-8",
    )


def run_windows_batch(
    script: Path,
    arguments: list[str],
    *,
    cwd: Path,
    env: dict[str, str],
) -> subprocess.CompletedProcess:
    command = "call {} {}".format(
        subprocess.list2cmdline([str(script)]),
        subprocess.list2cmdline(arguments),
    ).strip()
    return subprocess.run(
        [
            os.environ.get("COMSPEC", "cmd.exe"),
            "/d",
            "/s",
            "/c",
            command,
        ],
        cwd=str(cwd),
        env=env,
        capture_output=True,
        text=True,
        encoding="utf-8",
    )


class SkillsCliTests(unittest.TestCase):
    def run_cli(self, project: Path, source: Path, *args: str) -> subprocess.CompletedProcess:
        env = os.environ.copy()
        env["AIW_SKILLS_SOURCE_ROOT"] = str(source)
        return subprocess.run(
            [sys.executable, str(PLUGIN), *args],
            cwd=str(project),
            env=env,
            capture_output=True,
            text=True,
            encoding="utf-8",
        )

    def test_help_explains_commands_and_default_target(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)

            result = self.run_cli(project, source, "--help")

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("Manage canonical AIW Skills", result.stdout)
            self.assertIn("list", result.stdout)
            self.assertIn("install", result.stdout)
            self.assertIn(".agents/skills", result.stdout)
            self.assertIn("Quick start", result.stdout)
            self.assertIn("aiw skills install tdd --dry-run", result.stdout)

    def test_command_help_explains_use_constraints_and_examples(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)

            list_help = self.run_cli(project, source, "list", "--help")
            install_help = self.run_cli(project, source, "install", "--help")

            self.assertEqual(list_help.returncode, 0, list_help.stderr)
            self.assertIn("Use this when", list_help.stdout)
            self.assertIn("aiw skills list --json", list_help.stdout)
            self.assertEqual(install_help.returncode, 0, install_help.stderr)
            self.assertIn("Use this when", install_help.stdout)
            self.assertIn("unmanaged", install_help.stdout)
            self.assertIn("aiw skills install tdd --dry-run", install_help.stdout)

    def test_list_prints_valid_skills_in_name_order(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            write_skill(source, "zeta-review", "Review the final result.")
            write_skill(source, "alpha-plan", "Plan one focused change.")

            result = self.run_cli(project, source, "list")

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(
                result.stdout,
                "Canonical Skills:\n"
                "  alpha-plan - Plan one focused change.\n"
                "  zeta-review - Review the final result.\n",
            )
            self.assertEqual(result.stderr, "")
            self.assertFalse((project / ".agents").exists())

    def test_list_uses_root_skills_in_packaged_layout(self):
        with TemporaryDirectory() as temp_dir:
            base = Path(temp_dir)
            install_root = base / "install"
            plugin_dir = install_root / "plugins" / "aiw-skills"
            plugin_dir.mkdir(parents=True)
            packaged_plugin = plugin_dir / "aiw-skills.py"
            shutil.copy2(str(PLUGIN), str(packaged_plugin))
            source = install_root / "skills"
            source.mkdir()
            write_skill(source, "packaged-skill", "Read from the release root.")
            project = base / "project"
            project.mkdir()
            env = os.environ.copy()
            env.pop("AIW_SKILLS_SOURCE_ROOT", None)

            result = subprocess.run(
                [sys.executable, str(packaged_plugin), "list", "--json"],
                cwd=str(project),
                env=env,
                capture_output=True,
                text=True,
                encoding="utf-8",
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            payload = json.loads(result.stdout)
            self.assertEqual(
                [skill["name"] for skill in payload["skills"]],
                ["packaged-skill"],
            )
            self.assertEqual(
                Path(payload["skills"][0]["source"]),
                source / "packaged-skill",
            )

    @unittest.skipUnless(os.name == "nt", "Windows release script")
    def test_install_plugin_script_packages_root_skills_for_cli(self):
        for arguments in ([], ["aiw-skills"]):
            with self.subTest(arguments=arguments), TemporaryDirectory() as temp_dir:
                base = Path(temp_dir)
                script, install_root, env = make_windows_release_fixture(base)

                installed = run_windows_batch(
                    script,
                    arguments,
                    cwd=script.parent,
                    env=env,
                )

                self.assertEqual(installed.returncode, 0, installed.stderr)
                packaged_plugin = (
                    install_root / "plugins" / "aiw-skills" / "aiw-skills.py"
                )
                listed = subprocess.run(
                    [sys.executable, str(packaged_plugin), "list", "--json"],
                    cwd=str(base),
                    env=env,
                    capture_output=True,
                    text=True,
                    encoding="utf-8",
                )
                self.assertEqual(listed.returncode, 0, listed.stderr)
                payload = json.loads(listed.stdout)
                self.assertEqual(
                    [skill["name"] for skill in payload["skills"]],
                    ["packaged-skill"],
                )

    @unittest.skipUnless(os.name == "nt", "Windows release script")
    def test_install_plugin_script_propagates_skill_copy_failure(self):
        with TemporaryDirectory() as temp_dir:
            base = Path(temp_dir)
            script, _, env = make_windows_release_fixture(base)
            make_skill_copy_fail(env)

            result = run_windows_batch(
                script,
                ["aiw-skills"],
                cwd=script.parent,
                env=env,
            )

            copy_log = Path(env["COPY_LOG"])
            self.assertTrue(
                copy_log.exists(),
                result.stdout + result.stderr,
            )
            self.assertIn("skills", copy_log.read_text(encoding="utf-8"))
            self.assertNotEqual(
                result.returncode,
                0,
                result.stdout + result.stderr,
            )

    @unittest.skipUnless(os.name == "nt", "Windows release script")
    def test_build_script_propagates_skill_copy_failure(self):
        with TemporaryDirectory() as temp_dir:
            base = Path(temp_dir)
            _, install_root, env = make_windows_release_fixture(base)
            source_root = base / "release-source"
            script = source_root / "build.bat"
            script.write_text(
                BUILD_SCRIPT.read_text(encoding="utf-8").replace(
                    "set INSTALL_DIR=c:\\green\\aiw",
                    'set "INSTALL_DIR={}"'.format(install_root),
                ),
                encoding="utf-8",
            )
            make_skill_copy_fail(env)

            result = run_windows_batch(
                script,
                ["plugins"],
                cwd=source_root,
                env=env,
            )

            copy_log = Path(env["COPY_LOG"])
            self.assertTrue(
                copy_log.exists(),
                result.stdout + result.stderr,
            )
            self.assertIn("skills", copy_log.read_text(encoding="utf-8"))
            self.assertNotEqual(
                result.returncode,
                0,
                result.stdout + result.stderr,
            )

    def test_list_reports_invalid_metadata_without_listing_candidate(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            write_skill(source, "valid-skill", "A valid Skill.")
            invalid = source / "invalid-skill"
            invalid.mkdir()
            (invalid / "SKILL.md").write_text(
                "---\nname: invalid-skill\n---\n",
                encoding="utf-8",
            )

            result = self.run_cli(project, source, "list")

            self.assertEqual(result.returncode, 0)
            self.assertIn("valid-skill - A valid Skill.", result.stdout)
            self.assertNotIn("invalid-skill -", result.stdout)
            self.assertIn(str(invalid), result.stderr)
            self.assertIn("frontmatter description", result.stderr)
            self.assertFalse((project / ".agents").exists())

    def test_list_rejects_folder_and_declared_name_mismatch(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            mismatched = write_skill(
                source,
                "folder-name",
                "Do not list a mismatched Skill.",
            )
            skill_md = mismatched / "SKILL.md"
            skill_md.write_bytes(
                skill_md.read_bytes().replace(b"name: folder-name", b"name: other-name")
            )

            result = self.run_cli(project, source, "list")

            self.assertEqual(result.returncode, 0)
            self.assertNotIn("other-name -", result.stdout)
            self.assertIn("does not match folder", result.stderr)

    def test_install_copies_complete_skill_to_default_project_target(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            skill = write_skill(source, "review-work", "Review one change.")
            references = skill / "references"
            references.mkdir()
            (references / "checklist.md").write_bytes(b"# Review checklist\n")

            result = self.run_cli(project, source, "install", "review-work")

            installed = project / ".agents" / "skills" / "review-work"
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("Installed review-work", result.stdout)
            self.assertEqual(result.stderr, "")
            self.assertEqual(
                (installed / "SKILL.md").read_text(encoding="utf-8"),
                (skill / "SKILL.md").read_text(encoding="utf-8"),
            )
            self.assertEqual(
                (installed / "references" / "checklist.md").read_text(
                    encoding="utf-8"
                ),
                "# Review checklist\n",
            )
            self.assertEqual(
                list((project / ".agents" / "skills").glob(".aiw-stage-*")),
                [],
            )
            manifest = json.loads(
                (project / ".agents" / "skills" / ".aiw-skills.json").read_text(
                    encoding="utf-8"
                )
            )
            self.assertEqual(manifest["schema_version"], 1)
            self.assertEqual(
                manifest["skills"]["review-work"],
                {
                    "mode": "copy",
                    "sha256": (
                        "871a0f5a05bca1eec2cb45c6816f6ecd"
                        "8ecfef3ac72883877d80ff65a0a57b75"
                    ),
                    "source_identity": str(skill.resolve()),
                    "source_revision": None,
                },
            )
            self.assertEqual(
                list((project / ".agents" / "skills").glob(".aiw-manifest-*")),
                [],
            )

    def test_install_copies_shared_work_management_reference_when_declared(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            skill = write_skill(source, "shared-contract", "Use the shared contract.")
            skill_md = skill / "SKILL.md"
            skill_md.write_text(
                skill_md.read_text(encoding="utf-8") + "\nRead `skills/work-management.md`.\n",
                encoding="utf-8",
            )

            result = self.run_cli(project, source, "install", "shared-contract")

            installed = project / ".agents" / "skills" / "shared-contract"
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue((installed / "skills" / "work-management.md").is_file())
            self.assertEqual(
                (installed / "skills" / "work-management.md").read_text(
                    encoding="utf-8"
                ),
                (Path(__file__).resolve().parents[3] / "skills" / "work-management.md").read_text(
                    encoding="utf-8"
                ),
            )
            self.assertIn("Installed shared-contract", result.stdout)
            self.assertEqual(result.stderr, "")

    def test_install_rejects_invalid_metadata_before_writing(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            invalid = source / "broken-skill"
            invalid.mkdir()
            (invalid / "SKILL.md").write_text(
                "---\nname: broken-skill\n---\n",
                encoding="utf-8",
            )

            result = self.run_cli(project, source, "install", "broken-skill")

            self.assertEqual(result.returncode, 1)
            self.assertIn("frontmatter description", result.stderr)
            self.assertFalse((project / ".agents").exists())

    def test_install_rejects_symlink_before_writing(self):
        with TemporaryDirectory() as temp_dir:
            base, source, project = make_workspace(temp_dir)
            skill = write_skill(source, "linked-skill", "Reject linked content.")
            target = base / "outside.txt"
            target.write_text("outside\n", encoding="utf-8")
            link = skill / "outside-link.txt"
            try:
                os.symlink(str(target), str(link))
            except OSError as exc:
                self.skipTest("File symlinks are unavailable: {}".format(exc))

            result = self.run_cli(project, source, "install", "linked-skill")

            self.assertEqual(result.returncode, 1)
            self.assertIn("Unsupported symlink", result.stderr)
            self.assertFalse((project / ".agents").exists())

    def test_install_rejects_root_skill_symlink_before_writing(self):
        with TemporaryDirectory() as temp_dir:
            base, source, project = make_workspace(temp_dir)
            target_root = base / "target"
            target = write_skill(target_root, "root-link", "Reject the root link.")
            link = source / "root-link"
            try:
                os.symlink(str(target), str(link), target_is_directory=True)
            except OSError as exc:
                self.skipTest("Directory symlinks are unavailable: {}".format(exc))

            result = self.run_cli(project, source, "install", "root-link")

            self.assertEqual(result.returncode, 1)
            self.assertIn("Unsupported symlink", result.stderr)
            self.assertFalse((project / ".agents").exists())

    def test_install_rejects_unsupported_frontmatter_scalar(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            invalid = source / "invalid-scalar"
            invalid.mkdir()
            (invalid / "SKILL.md").write_bytes(
                b"---\n"
                b"name: invalid-scalar\n"
                b'description: "missing close quote\n'
                b"---\n"
            )

            result = self.run_cli(project, source, "install", "invalid-scalar")

            self.assertEqual(result.returncode, 1)
            self.assertIn("frontmatter description", result.stderr)
            self.assertFalse((project / ".agents").exists())

            (invalid / "SKILL.md").write_bytes(
                b"---\n"
                b"name: invalid-scalar\n"
                b'description: "closed" trailing"\n'
                b"---\n"
            )

            result = self.run_cli(project, source, "install", "invalid-scalar")

            self.assertEqual(result.returncode, 1)
            self.assertIn("frontmatter description", result.stderr)
            self.assertFalse((project / ".agents").exists())

    def test_list_reports_empty_frontmatter_scalar_without_traceback(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            invalid = source / "empty-description"
            invalid.mkdir()
            (invalid / "SKILL.md").write_bytes(
                b"---\n"
                b"name: empty-description\n"
                b"description:\n"
                b"---\n"
            )

            result = self.run_cli(project, source, "list", "--json")

            self.assertEqual(result.returncode, 0, result.stderr)
            payload = json.loads(result.stdout)
            self.assertEqual(payload["skills"], [])
            self.assertIn("frontmatter description", payload["issues"][0])
            self.assertNotIn("Traceback", result.stderr)

    def test_install_rejects_comment_only_frontmatter_scalar_as_json(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            invalid = source / "comment-description"
            invalid.mkdir()
            (invalid / "SKILL.md").write_bytes(
                b"---\n"
                b"name: comment-description\n"
                b"description: # missing value\n"
                b"---\n"
            )

            result = self.run_cli(
                project,
                source,
                "install",
                "comment-description",
                "--json",
            )

            self.assertEqual(result.returncode, 1, result.stderr)
            payload = json.loads(result.stdout)
            self.assertFalse(payload["ok"])
            self.assertIn("frontmatter description", payload["error"])
            self.assertEqual(result.stderr, "")
            self.assertFalse((project / ".agents").exists())

    def test_list_accepts_colon_and_hash_inside_quoted_scalar(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            skill = source / "quoted-description"
            skill.mkdir()
            (skill / "SKILL.md").write_bytes(
                b"---\n"
                b"name: quoted-description\n"
                b'description: "Reason: use # safely"\n'
                b"---\n"
            )

            result = self.run_cli(project, source, "list", "--json")

            self.assertEqual(result.returncode, 0, result.stderr)
            payload = json.loads(result.stdout)
            self.assertEqual(len(payload["skills"]), 1)
            self.assertEqual(
                payload["skills"][0]["description"],
                "Reason: use # safely",
            )
            self.assertEqual(payload["skills"][0]["name"], "quoted-description")
            self.assertEqual(payload["issues"], [])

    def test_install_dry_run_reports_plan_without_writing(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            skill = write_skill(source, "plan-only", "Preview this install.")

            result = self.run_cli(
                project,
                source,
                "install",
                "plan-only",
                "--dry-run",
            )

            destination = project / ".agents" / "skills" / "plan-only"
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("Would install plan-only", result.stdout)
            self.assertIn(str(skill.resolve()), result.stdout)
            self.assertIn(str(destination), result.stdout)
            self.assertEqual(result.stderr, "")
            self.assertFalse((project / ".agents").exists())

    def test_install_preserves_unmanaged_same_name_destination(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            write_skill(source, "owned-by-user", "Do not replace this.")
            destination = (
                project / ".agents" / "skills" / "owned-by-user"
            )
            destination.mkdir(parents=True)
            sentinel = destination / "user.txt"
            sentinel.write_text("keep me", encoding="utf-8")

            result = self.run_cli(
                project,
                source,
                "install",
                "owned-by-user",
            )

            self.assertEqual(result.returncode, 1)
            self.assertIn("unmanaged", result.stderr.lower())
            self.assertEqual(sentinel.read_text(encoding="utf-8"), "keep me")
            self.assertFalse(
                (project / ".agents" / "skills" / ".aiw-skills.json").exists()
            )

    def test_discover_reports_managed_and_unmanaged_skills(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            write_skill(source, "managed-skill", "Install first.")
            write_skill(source, "unmanaged-skill", "Leave unmanaged.")

            managed_result = self.run_cli(project, source, "install", "managed-skill")
            self.assertEqual(managed_result.returncode, 0, managed_result.stderr)

            unmanaged_dir = project / ".agents" / "skills" / "unmanaged-skill"
            unmanaged_dir.mkdir(parents=True)
            (unmanaged_dir / "SKILL.md").write_text(
                "---\nname: unmanaged-skill\ndescription: \"Leave unmanaged.\"\n---\n",
                encoding="utf-8",
            )

            result = self.run_cli(project, source, "discover", "--json")

            self.assertEqual(result.returncode, 0, result.stderr)
            payload = json.loads(result.stdout)
            self.assertEqual(payload["action"], "discover")
            self.assertEqual(
                {skill["name"]: skill["status"] for skill in payload["skills"]},
                {
                    "managed-skill": "managed",
                    "unmanaged-skill": "unmanaged",
                },
            )

    def test_discover_json_is_one_machine_readable_result(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            write_skill(source, "discover-json", "Discover as JSON.")

            install = self.run_cli(project, source, "install", "discover-json")
            self.assertEqual(install.returncode, 0, install.stderr)

            result = self.run_cli(project, source, "discover", "--json")

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stderr, "")
            payload = json.loads(result.stdout)
            self.assertEqual(payload["action"], "discover")
            self.assertEqual(payload["root"], str(project / ".agents" / "skills"))
            self.assertEqual(payload["issues"], [])
            self.assertEqual(len(payload["skills"]), 1)
            self.assertEqual(payload["skills"][0]["name"], "discover-json")
            self.assertEqual(payload["skills"][0]["status"], "managed")
            self.assertRegex(payload["skills"][0]["sha256"], r"^[0-9a-f]{64}$")

    def test_adopt_records_existing_unmanaged_skill(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            write_skill(source, "ask-matt", "Adopt this Skill.")
            destination = project / ".agents" / "skills" / "ask-matt"
            destination.mkdir(parents=True)
            (destination / "SKILL.md").write_text(
                "---\nname: ask-matt\ndescription: \"Adopt this Skill.\"\n---\n",
                encoding="utf-8",
            )

            adopt = self.run_cli(project, source, "adopt", "--json")
            self.assertEqual(adopt.returncode, 0, adopt.stderr)
            adopt_payload = json.loads(adopt.stdout)
            self.assertEqual(adopt_payload["adopted"], ["ask-matt"])
            self.assertTrue((project / ".agents" / "skills" / ".aiw-skills.json").exists())

            reinstall = self.run_cli(project, source, "install", "ask-matt")
            self.assertEqual(reinstall.returncode, 0, reinstall.stderr)
            self.assertIn("Installed ask-matt", reinstall.stdout)

    def test_install_preserves_unmanaged_dangling_destination_symlink(self):
        with TemporaryDirectory() as temp_dir:
            base, source, project = make_workspace(temp_dir)
            write_skill(source, "dangling-skill", "Protect a dangling link.")
            destination = (
                project / ".agents" / "skills" / "dangling-skill"
            )
            destination.parent.mkdir(parents=True)
            missing_target = base / "missing-target"
            try:
                os.symlink(
                    str(missing_target),
                    str(destination),
                    target_is_directory=True,
                )
            except OSError as exc:
                self.skipTest("Directory symlinks are unavailable: {}".format(exc))

            result = self.run_cli(
                project,
                source,
                "install",
                "dangling-skill",
            )

            self.assertEqual(result.returncode, 1)
            self.assertIn("unmanaged", result.stderr.lower())
            self.assertTrue(destination.is_symlink())
            self.assertFalse(missing_target.exists())

    def test_identical_managed_reinstall_is_no_op(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            write_skill(source, "stable-skill", "Install once.")

            first = self.run_cli(project, source, "install", "stable-skill")
            self.assertEqual(first.returncode, 0, first.stderr)
            destination = (
                project / ".agents" / "skills" / "stable-skill" / "SKILL.md"
            )
            manifest_path = (
                project / ".agents" / "skills" / ".aiw-skills.json"
            )
            installed_mtime = destination.stat().st_mtime_ns
            manifest_bytes = manifest_path.read_bytes()
            manifest_mtime = manifest_path.stat().st_mtime_ns

            second = self.run_cli(project, source, "install", "stable-skill")

            self.assertEqual(second.returncode, 0, second.stderr)
            self.assertIn("Already installed stable-skill", second.stdout)
            self.assertEqual(second.stderr, "")
            self.assertEqual(destination.stat().st_mtime_ns, installed_mtime)
            self.assertEqual(manifest_path.read_bytes(), manifest_bytes)
            self.assertEqual(manifest_path.stat().st_mtime_ns, manifest_mtime)

    def test_changed_managed_install_can_be_reinstalled(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            write_skill(source, "changed-skill", "Protect local changes.")
            first = self.run_cli(project, source, "install", "changed-skill")
            self.assertEqual(first.returncode, 0, first.stderr)
            installed = (
                project / ".agents" / "skills" / "changed-skill" / "local.txt"
            )
            installed.write_text("local change", encoding="utf-8")

            second = self.run_cli(project, source, "install", "changed-skill")

            self.assertEqual(second.returncode, 0, second.stderr)
            self.assertIn("Installed changed-skill", second.stdout)
            self.assertFalse(installed.exists())

    def test_sync_requires_managed_destination_and_republishes_content(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            skill = write_skill(source, "sync-skill", "Republish me.")

            install = self.run_cli(project, source, "install", "sync-skill")
            self.assertEqual(install.returncode, 0, install.stderr)

            changed = project / ".agents" / "skills" / "sync-skill" / "local.txt"
            changed.write_text("local change", encoding="utf-8")

            sync = self.run_cli(project, source, "sync", "sync-skill")
            self.assertEqual(sync.returncode, 0, sync.stderr)
            self.assertIn("Installed sync-skill", sync.stdout)
            self.assertFalse(changed.exists())
            self.assertEqual(
                (project / ".agents" / "skills" / "sync-skill" / "SKILL.md").read_text(
                    encoding="utf-8"
                ),
                (skill / "SKILL.md").read_text(encoding="utf-8"),
            )

            sync_json = self.run_cli(project, source, "sync", "sync-skill", "--json")
            self.assertEqual(sync_json.returncode, 0, sync_json.stderr)
            self.assertEqual(sync_json.stderr, "")
            payload = json.loads(sync_json.stdout)
            self.assertEqual(payload["action"], "install")
            self.assertEqual(payload["status"], "installed")
            self.assertEqual(payload["name"], "sync-skill")
            self.assertRegex(payload["sha256"], r"^[0-9a-f]{64}$")

    def test_list_json_is_one_machine_readable_result(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            skill = write_skill(source, "json-skill", "List 鈥?as JSON.")

            result = self.run_cli(project, source, "list", "--json")

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stderr, "")
            self.assertEqual(
                json.loads(result.stdout),
                {
                    "action": "list",
                    "issues": [],
                    "ok": True,
                    "skills": [
                        {
                            "description": "List 鈥?as JSON.",
                            "name": "json-skill",
                            "source": str(skill.resolve()),
                        }
                    ],
                },
            )

    def test_install_json_reports_dry_run_and_installed_digest(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)
            write_skill(source, "json-install", "Install as JSON.")
            destination = project / ".agents" / "skills" / "json-install"

            preview = self.run_cli(
                project,
                source,
                "install",
                "json-install",
                "--dry-run",
                "--json",
            )

            self.assertEqual(preview.returncode, 0, preview.stderr)
            self.assertEqual(preview.stderr, "")
            preview_data = json.loads(preview.stdout)
            self.assertEqual(preview_data["status"], "would_install")
            self.assertEqual(preview_data["destination"], str(destination))
            self.assertFalse((project / ".agents").exists())

            installed = self.run_cli(
                project,
                source,
                "install",
                "json-install",
                "--json",
            )

            self.assertEqual(installed.returncode, 0, installed.stderr)
            self.assertEqual(installed.stderr, "")
            installed_data = json.loads(installed.stdout)
            self.assertEqual(installed_data["status"], "installed")
            self.assertEqual(installed_data["name"], "json-install")
            self.assertRegex(installed_data["sha256"], r"^[0-9a-f]{64}$")

    def test_install_json_operational_error_uses_stdout_only(self):
        with TemporaryDirectory() as temp_dir:
            _, source, project = make_workspace(temp_dir)

            result = self.run_cli(
                project,
                source,
                "install",
                "missing-skill",
                "--json",
            )

            self.assertEqual(result.returncode, 1)
            self.assertEqual(result.stderr, "")
            self.assertEqual(
                json.loads(result.stdout),
                {
                    "action": "install",
                    "error": "Canonical Skill not found: missing-skill",
                    "ok": False,
                },
            )


if __name__ == "__main__":
    unittest.main()
