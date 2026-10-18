"""Build one explicitly named AIW package."""
from pathlib import Path
import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from package_definitions import run

if __name__ == "__main__":
    run('skill-domain-modeling')
