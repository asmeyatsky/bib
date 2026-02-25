#!/usr/bin/env python3
"""Update service main.go files to pass logger to gRPC handlers."""
import glob
import re

def update_main_file(filepath):
    """Update a service's main.go to pass logger to handler."""
    with open(filepath, 'r') as f:
        content = f.read()
    
    original = content
    
    # Find New{Handler} calls and add logger parameter
    for handler in ['NewIdentityHandler', 'NewFXHandler', 'NewLedgerHandler', 
                    'NewAccountHandler', 'NewPaymentHandler', 'NewDepositHandler',
                    'NewCardHandler', 'NewLendingHandler', 'NewFraudHandler',
                    'NewReportingHandler']:
        # Pattern to match handler creation
        pattern = rf'({handler}\(\s*[^)]+)(\))'
        match = re.search(pattern, content)
        if match:
            args = match.group(1)
            # Check if logger is already passed
            if 'logger' not in args.lower():
                # Add logger as last argument
                if args.rstrip().endswith(','):
                    new_args = args + '\n\t\tlogger,'
                else:
                    new_args = args + ',\n\t\tlogger,'
                content = content.replace(args, new_args)
    
    if content != original:
        with open(filepath, 'w') as f:
            f.write(content)
        return True
    return False

# Update all service main.go files
count = 0
for f in glob.glob('services/*/cmd/*/main.go'):
    if update_main_file(f):
        print(f'✓ Updated {f}')
        count += 1
    else:
        print(f'- No changes needed: {f}')

print(f'\nTotal: {count} files updated')
