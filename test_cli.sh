#!/bin/bash

echo "Testing CLI functionality..."

# Test CLI with expected commands
echo -e "try me\nunsupported command\n/end" | ./bin/repo-explanation -mode=cli

echo -e "\n--- Test completed ---"
