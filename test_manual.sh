#!/bin/bash

echo "Testing autocomplete functionality..."
echo "Checking if backend server is running..."

# Test if server is up
if ! curl -s --connect-timeout 5 http://localhost:5173 > /dev/null; then
    echo "❌ Frontend server not responding"
    exit 1
fi

if ! curl -s --connect-timeout 5 http://localhost:8080 > /dev/null 2>&1; then
    echo "❌ Backend server not responding"
    exit 1
fi

echo "✅ Servers are running"
echo
echo "Manual test instructions:"
echo "1. Open browser at http://localhost:5173"
echo "2. Look for 'Bröd 🍞' item with 'Bröd & Bakverk' category badge"
echo "3. Type 'Br' in the input field at the bottom"
echo "4. Autocomplete dropdown should show:"
echo "   - 'Bröd 🍞' with 'Bröd & Bakverk' category badge"
echo
echo "Expected behavior:"
echo "- When typing 'Bröd', the autocomplete should show the category"
echo "- When pressing Enter, the new todo should be assigned to 'Bröd & Bakverk' category"
echo
echo "If category doesn't show in autocomplete, check backend logs for autocomplete requests"