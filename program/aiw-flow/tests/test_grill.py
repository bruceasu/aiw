import unittest

from codex_flow.grill import GRILL_INSTRUCTIONS, build_initial_grill_prompt


class GrillTests(unittest.TestCase):
    def test_instructions_define_interview_and_completion_contract(self):
        self.assertIn("Ask at most one user decision question", GRILL_INSTRUCTIONS)
        self.assertIn("Recommendation", GRILL_INSTRUCTIONS)
        self.assertIn("SUCCESS: Ready to execute.", GRILL_INSTRUCTIONS)
        self.assertIn("Do not implement", GRILL_INSTRUCTIONS)

    def test_initial_prompt_contains_requirement_and_context(self):
        prompt = build_initial_grill_prompt("Add CSV export.", "# Workspace Context\nGo project")

        self.assertIn("Add CSV export.", prompt)
        self.assertIn("Go project", prompt)

    def test_initial_prompt_rejects_empty_requirement(self):
        with self.assertRaises(ValueError):
            build_initial_grill_prompt(" \n", "context")
