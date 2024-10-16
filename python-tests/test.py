import base64
import json
import time
import requests
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

PRIVATE_KEY_BASE64 = "6sMrTQoXAXEhv3gsURcRJSwDVdU4lJtiLX5jWPf7yiU=" # Testing key, safe for version control
PRIVATE_KEY_BYTES = base64.b64decode(PRIVATE_KEY_BASE64)
private_key = Ed25519PrivateKey.from_private_bytes(PRIVATE_KEY_BYTES)

BASE_URL = "http://localhost:8082"


def sign_data(data):
    data_string = json.dumps(data, separators=(',', ':'), sort_keys=True)
    signature = private_key.sign(data_string.encode('utf-8'))
    return base64.b64encode(signature).decode('utf-8'), data_string


def send_request(endpoint, data):
    signature, data_string = sign_data(data)
    payload = {
        "data": data_string,
        "signature": signature
    }
    print(f"Sending request to {endpoint} with payload: {json.dumps(payload)}")
    url = BASE_URL + endpoint
    response = requests.post(url, json=payload, headers={"Content-Type": "application/json"})
    return response


def test_add_role():
    data = {
        "timestamp": int(time.time() * 1000),
        "userID": "938762267049218128",
        "roleID": "1293551323429470310"
    }
    response = send_request("/addrole", data)
    print("Add Role Test:")
    print(f"Status: {response.status_code}, Response: {response.text}\n")


def test_remove_role():
    data = {
        "timestamp": int(time.time() * 1000),
        "userID": "938762267049218128",
        "roleID": "1293551323429470310"
    }
    response = send_request("/removerole", data)
    print("Remove Role Test:")
    print(f"Status: {response.status_code}, Response: {response.text}\n")


def test_send_message():
    data = {
        "timestamp": int(time.time() * 1000),
        "channelID": "1128989517785862164",
        "message": "Hello, this is a test message."
    }
    response = send_request("/sendmessage", data)
    print("Send Message Test:")
    print(f"Status: {response.status_code}, Response: {response.text}\n")


def test_dm_user():
    data = {
        "timestamp": int(time.time() * 1000),
        "userID": "938762267049218128",
        "message": "Hello, this is a DM test message."
    }
    response = send_request("/dmuser", data)
    print("DM User Test:")
    print(f"Status: {response.status_code}, Response: {response.text}\n")


def test_invalid_signature():
    data = {
        "timestamp": int(time.time() * 1000),
        "userID": "938762267049218128",
        "roleID": "1293551323429470310"
    }
    payload = {
        "data": data,
        "signature": "InvalidSignature"
    }
    url = BASE_URL + "/addrole"
    response = requests.post(url, json=payload, headers={"Content-Type": "application/json"})
    print("Invalid Signature Test:")
    print(f"Status: {response.status_code}, Response: {response.text}\n")


def test_replay_attack():
    old_timestamp = int(time.time() * 1000) - 60000
    data = {
        "timestamp": old_timestamp,
        "userID": "938762267049218128",
        "roleID": "1293551323429470310"
    }
    response = send_request("/addrole", data)
    print("Replay Attack Test:")
    print(f"Status: {response.status_code}, Response: {response.text}\n")


def test_invalid_request():
    data = {
        "timestamp": int(time.time() * 1000)
        # Missing required fields
    }
    response = send_request("/addrole", data)
    print("Invalid Request Test:")
    print(f"Status: {response.status_code}, Response: {response.text}\n")


if __name__ == "__main__":
    test_add_role()
    test_remove_role()
    test_send_message()
    test_dm_user()
    test_invalid_signature()
    test_replay_attack()
    test_invalid_request()
