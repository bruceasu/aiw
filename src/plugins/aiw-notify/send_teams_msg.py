"""Source-tree entry; packaging replaces this with the complete Teams sender."""
from pathlib import Path
import runpy

if __name__ == "__main__":
    runpy.run_path(str(Path(__file__).resolve().parents[1] / "send_teams_msg.py"), run_name="__main__")
