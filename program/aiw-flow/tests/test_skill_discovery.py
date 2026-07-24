import os
import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from codex_flow.skill_discovery import (
    SkillMetadataError,
    discover_skills,
    read_skill_metadata,
)


def write_skill(root: Path, folder: str, name: str, description: str) -> Path:
    skill_dir = root / folder
    skill_dir.mkdir(parents=True)
    (skill_dir / "SKILL.md").write_text(
        "---\n"
        "name: {}\n"
        'description: "{}"\n'
        "---\n\n"
        "Follow this workflow.\n".format(name, description),
        encoding="utf-8",
    )
    return skill_dir


class SkillDiscoveryTests(unittest.TestCase):
    def test_discovers_repository_ancestry_and_user_scopes(self):
        with TemporaryDirectory() as temp_dir:
            base = Path(temp_dir)
            repo = base / "repo"
            workspace = repo / "services" / "billing"
            workspace.mkdir(parents=True)
            (repo / ".git").mkdir()
            user_home = base / "home"
            codex_home = base / "codex-home"

            write_skill(
                workspace / ".agents" / "skills",
                "local",
                "local-review",
                "Review the local service.",
            )
            write_skill(
                repo / ".agents" / "skills",
                "shared",
                "shared-review",
                "Review the repository.",
            )
            write_skill(
                repo / ".codex" / "skills",
                "legacy-project",
                "legacy-project",
                "Use the aiw project convention.",
            )
            write_skill(
                user_home / ".agents" / "skills",
                "user",
                "user-review",
                "Review any project.",
            )
            write_skill(
                codex_home / "skills",
                "legacy-user",
                "legacy-user",
                "Use the configured Codex home.",
            )

            result = discover_skills(
                workspace,
                codex_home=codex_home,
                user_home=user_home,
            )

            self.assertEqual(
                [skill.name for skill in result.skills],
                [
                    "local-review",
                    "shared-review",
                    "legacy-project",
                    "user-review",
                    "legacy-user",
                ],
            )
            self.assertEqual(
                [skill.scope for skill in result.skills],
                ["project", "project", "project", "user", "user"],
            )
            self.assertEqual(result.issues, ())

    def test_non_git_workspace_is_project_root(self):
        with TemporaryDirectory() as temp_dir:
            base = Path(temp_dir)
            workspace = base / "workspace"
            workspace.mkdir()
            write_skill(
                workspace / ".codex" / "skills",
                "project",
                "project-review",
                "Review this project.",
            )

            result = discover_skills(
                workspace,
                user_home=base / "empty-home",
            )

            self.assertEqual([skill.name for skill in result.skills], ["project-review"])

    def test_reports_malformed_metadata_and_keeps_valid_skills(self):
        with TemporaryDirectory() as temp_dir:
            base = Path(temp_dir)
            workspace = base / "workspace"
            skills_root = workspace / ".agents" / "skills"
            workspace.mkdir()
            write_skill(skills_root, "valid", "valid-skill", "A valid Skill.")
            invalid = skills_root / "invalid"
            invalid.mkdir()
            (invalid / "SKILL.md").write_text(
                "---\nname: Invalid Name\n---\n",
                encoding="utf-8",
            )

            result = discover_skills(
                workspace,
                user_home=base / "empty-home",
            )

            self.assertEqual([skill.name for skill in result.skills], ["valid-skill"])
            self.assertEqual(len(result.issues), 1)
            self.assertEqual(result.issues[0].source, invalid)
            self.assertIn("invalid frontmatter name", result.issues[0].message)

    def test_groups_duplicate_names_without_selecting_precedence(self):
        with TemporaryDirectory() as temp_dir:
            base = Path(temp_dir)
            workspace = base / "workspace"
            workspace.mkdir()
            write_skill(
                workspace / ".agents" / "skills",
                "project",
                "duplicate",
                "Project copy.",
            )
            write_skill(
                base / "home" / ".agents" / "skills",
                "user",
                "duplicate",
                "User copy.",
            )

            result = discover_skills(workspace, user_home=base / "home")

            self.assertEqual(len(result.by_name()["duplicate"]), 2)

    def test_follows_symlinked_skill_directory(self):
        with TemporaryDirectory() as temp_dir:
            base = Path(temp_dir)
            workspace = base / "workspace"
            root = workspace / ".agents" / "skills"
            root.mkdir(parents=True)
            target = write_skill(
                base / "shared",
                "source",
                "linked-skill",
                "Use a linked Skill.",
            )
            link = root / "linked"
            try:
                os.symlink(str(target), str(link), target_is_directory=True)
            except OSError as exc:
                if os.name != "nt":
                    self.skipTest("Directory symlinks are unavailable: {}".format(exc))
                completed = subprocess.run(
                    ["cmd", "/c", "mklink", "/J", str(link), str(target)],
                    capture_output=True,
                    text=True,
                    encoding="utf-8",
                )
                if completed.returncode != 0:
                    self.skipTest(
                        "Directory links are unavailable: {}".format(
                            completed.stderr.strip() or completed.stdout.strip()
                        )
                    )

            result = discover_skills(
                workspace,
                user_home=base / "empty-home",
            )

            self.assertEqual([skill.name for skill in result.skills], ["linked-skill"])
            self.assertEqual(result.skills[0].source, target.resolve())

    def test_reads_quoted_metadata_and_rejects_block_description(self):
        with TemporaryDirectory() as temp_dir:
            base = Path(temp_dir)
            quoted = base / "quoted.md"
            quoted.write_text(
                "---\nname: 'quoted-skill'\ndescription: \"Quoted description.\"\n---\n",
                encoding="utf-8",
            )
            self.assertEqual(
                read_skill_metadata(quoted),
                ("quoted-skill", "Quoted description."),
            )

            block = base / "block.md"
            block.write_text(
                "---\nname: block-skill\ndescription: |\n  Multiline description.\n---\n",
                encoding="utf-8",
            )
            with self.assertRaisesRegex(SkillMetadataError, "description"):
                read_skill_metadata(block)


if __name__ == "__main__":
    unittest.main()
