import os
import sys

print(f"AIW_PLUGIN_NAME={os.environ.get('AIW_PLUGIN_NAME')}")
print(f"AIW_PLUGIN_PATH={os.environ.get('AIW_PLUGIN_PATH')}")
print(f"AIW_CMDLINE={os.environ.get('AIW_CMDLINE')}")
print("ARGS=", sys.argv[1:])
