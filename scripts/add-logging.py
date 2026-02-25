#!/usr/bin/env python3
"""Add error logging to all gRPC service handlers systematically."""
import glob
import re

def update_handler_file(filepath):
    """Add logging to a service's gRPC handler."""
    with open(filepath, 'r') as f:
        content = f.read()
    
    original = content
    
    # 1. Add slog import if missing
    if 'log/slog' not in content:
        content = content.replace(
            'import (\n\t"context"',
            'import (\n\t"context"\n\t"log/slog"'
        )
    
    # 2. Replace ALL TODO error comments with actual logging
    content = re.sub(
        r'// TODO: log original error server-side: err\s*\n\s*return nil, status\.Error\(codes\.Internal, "internal error"\)',
        'h.logger.Error("handler error", "error", err)\n\t\treturn nil, status.Error(codes.Internal, "internal error")',
        content
    )
    
    # 3. Add logger field to handler structs
    for handler in ['IdentityHandler', 'FXHandler', 'LedgerHandler', 'AccountHandler', 
                    'PaymentHandler', 'DepositHandler', 'CardHandler', 'LendingHandler', 
                    'FraudHandler', 'ReportingHandler']:
        # Add logger field
        pattern = rf'(type {handler} struct \{{[^}}]+)(}})'
        match = re.search(pattern, content, re.DOTALL)
        if match and 'logger' not in match.group(1):
            fields = match.group(1)
            if '\tlogger' not in fields and 'logger *' not in fields:
                new_fields = fields + '\n\tlogger               *slog.Logger'
                content = content.replace(fields, new_fields)
    
    # 4. Update NewHandler functions to accept logger parameter
    for handler in ['IdentityHandler', 'FXHandler', 'LedgerHandler', 'AccountHandler',
                    'PaymentHandler', 'DepositHandler', 'CardHandler', 'LendingHandler',
                    'FraudHandler', 'ReportingHandler']:
        # Find New{Handler} function and add logger param
        pattern = rf'(func New{handler}\([^)]+)(\) \*{handler})'
        match = re.search(pattern, content)
        if match and 'logger' not in match.group(1):
            params = match.group(1)
            # Add logger as last parameter
            if params.endswith(',\n'):
                new_params = params + '\tlogger *slog.Logger,\n'
            elif params.endswith(')'):
                new_params = params.replace(')', ',\n\tlogger *slog.Logger,)')
            else:
                new_params = params + ',\n\tlogger *slog.Logger'
            content = content.replace(params, new_params)
            
            # Update the struct initialization to include logger
            init_pattern = rf'(return &{handler}\{{[^}}]+)(}})'
            init_match = re.search(init_pattern, content, re.DOTALL)
            if init_match:
                init_fields = init_match.group(1)
                if 'logger:' not in init_fields and 'logger *' not in init_fields:
                    new_init = init_fields + '\n\t\tlogger:               logger,'
                    content = content.replace(init_fields, new_init)
    
    if content != original:
        with open(filepath, 'w') as f:
            f.write(content)
        return True
    return False

# Update all service handlers
count = 0
for f in glob.glob('services/*/internal/presentation/grpc/handler.go'):
    if update_handler_file(f):
        print(f'✓ Updated {f}')
        count += 1
    else:
        print(f'- No changes needed: {f}')

print(f'\nTotal: {count} files updated')
