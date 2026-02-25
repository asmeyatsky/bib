#!/usr/bin/env python3
import re

with open('docker-compose.yml', 'r') as f:
    content = f.read()

# Add start_period after interval for service healthchecks with wget
pattern = r'(test: \["CMD", "wget"[^]]*healthz"\]\n\s+interval: 10s)(?!.*start_period)'
replacement = r'\1\n      start_period: 30s'

new_content = re.sub(pattern, replacement, content, flags=re.DOTALL)

with open('docker-compose.yml', 'w') as f:
    f.write(new_content)

print("Added start_period to service healthchecks")
