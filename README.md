<p align="center">
  <img src="https://github.com/PlatinPay/.github/blob/main/horizontal-logo-text-transparent.png?raw=true" alt="PlatinPay's Logo" width="500"/>
</p>
<br />

# PlatinPay-Discord

[Find the main README here! (It has a showcase video and deployment links so you can try PlatinPay yourself!)](https://github.com/PlatinPay)

This is the Discord bot of the PlatinPay project.

## Features
- Modern, efficient design in Go.
- TOML config parsing for easy customization
- An API that allows the PlatinPay backend to:
    - Send messages
    - Add roles
    - Remove roles
    - DM players
- An aliased /shop command to let players get a link to your shop
- A variety of security features, such as the ability to only accept local inbound requests, and the ability to whitelist IPs making inbound requests.
- Ed25519 asymmetric signing

## Installation (Linux only ATM)
1. Download the 'platinpay-discord' binary from the releases section of this repository.
2. Put it in a folder alongside with a config.toml, look at the file in this repository for an example
4. Write DISCORD_TOKEN="YOUR TOKEN" to .env
5. Set your public key through the slash command
6. Execute the binary: `./platinpay-discord`

This project is licensed under the [GNU AGPL-3.0](LICENSE) License. See the `LICENSE` file for details.

**DISCLAIMER: This project is currently not ready for production usage. It is a work-in-progress.**
