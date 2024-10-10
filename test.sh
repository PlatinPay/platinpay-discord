#!/bin/bash

BASE_URL="http://localhost:8082"

USER_ID="938762267049218128"
CHANNEL_ID="1128989517785862164"
ROLE_ID="1293551323429470310"

counter=0

test_add_role() {
    echo "[] Testing /addrole endpoint..."
    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/addrole" \
        -d "userID=$USER_ID" \
        -d "roleID=$ROLE_ID")

    if [ "$response" -eq 200 ]; then
        echo "[✓] /addrole test passed."
        ((counter+=1))
    else
        echo "[✗] /addrole test failed. HTTP status code: $response"
    fi
}

test_remove_role() {
    echo "[] Testing /removerole endpoint..."
    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/removerole" \
        -d "userID=$USER_ID" \
        -d "roleID=$ROLE_ID")

    if [ "$response" -eq 200 ]; then
        echo "[✓] /removerole test passed."
        ((counter+=1))
    else
        echo "[✗] /removerole test failed. HTTP status code: $response"
    fi
}

test_send_message() {
    echo "[] Testing /sendmessage endpoint..."
    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/sendmessage" \
        -d "channelID=$CHANNEL_ID" \
        -d "message=Test message, <@$USER_ID>")

    if [ "$response" -eq 200 ]; then
        echo "[✓] /sendmessage test passed."
        ((counter+=1))
    else
        echo "[✗] /sendmessage test failed. HTTP status code: $response"
    fi
}

test_send_message() {
    echo "[] Testing /sendmessage endpoint..."
    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/sendmessage" \
        -d "channelID=$CHANNEL_ID" \
        -d "message=Test message, <@$USER_ID>")

    if [ "$response" -eq 200 ]; then
        echo "[✓] /sendmessage test passed."
        ((counter+=1))
    else
        echo "[✗] /sendmessage test failed. HTTP status code: $response"
    fi
}

test_dm_user() {
    echo "[] Testing /dmuser endpoint..."
    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/dmuser" \
        -d "userID=$USER_ID" \
        -d "message=Test message, <@$USER_ID>")

    if [ "$response" -eq 200 ]; then
        echo "[✓] /dmuser test passed."
        ((counter+=1))
    else
        echo "[✗] /dmuser test failed. HTTP status code: $response"
    fi
}

echo -e "Running tests on $BASE_URL with user ID $USER_ID...\n"
test_add_role
test_remove_role
test_send_message
test_dm_user
echo -e "\nTests completed. $counter/4 tests passed."