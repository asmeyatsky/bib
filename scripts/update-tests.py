#!/usr/bin/env python3
"""Update test files to pass logger to handler constructors."""
import glob
import re

count = 0
for f in glob.glob('services/*/*_test.go'):
    with open(f, 'r') as file:
        content = file.read()
    
    original = content
    
    # Add slog import if needed
    if 'log/slog' not in content and '"testing"' in content:
        content = content.replace('"testing"', '"log/slog"\n\t"testing"')
    
    # Find New*Handler calls and add logger parameter
    for handler in ['NewIdentityHandler', 'NewFXHandler', 'NewLedgerHandler', 
                    'NewAccountHandler', 'NewPaymentHandler', 'NewDepositHandler',
                    'NewCardHandler', 'NewLendingHandler', 'NewFraudHandler',
                    'NewReportingHandler']:
        # Pattern to match handler creation in tests
        pattern = rf'({handler}\(\s*[^)]+)(\))'
        matches = list(re.finditer(pattern, content))
        for match in reversed(matches):  # Reverse to not mess up positions
            args = match.group(1)
            # Check if logger is already passed
            if 'logger' not in args.lower() and 'log' not in args.lower():
                # Add logger as last argument
                if args.rstrip().endswith(','):
                    new_args = args + '\n\t\tlogger,'
                else:
                    new_args = args + ',\n\t\tlogger,'
                content = content[:match.start()] + new_args + content[match.end():]
    
    if content != original:
        with open(f, 'w') as file:
            file.write(content)
        print(f'✓ Updated {f}')
        count += 1

print(f'\nTotal: {count} files updated')
