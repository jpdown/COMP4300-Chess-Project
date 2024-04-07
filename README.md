# Chess, but like, low level networking

Written by Jaden Down.

## Description

This project implements a game of Chess, but operating using raw sockets at the Ethernet layer. This means that I do not have IP or TCP available to me. As such, I had to implement my own protocols for identifying machines, and for reliably transmitting data between two clients.

## Usage

This project has only been tested on Fedora 39, Fedora 40, and Ubuntu Server 20.04, using Go versions 1.20.14, and 1.18.1. It is highly unlikely that it will function at all in Windows. macOS may work, but I have developed it exclusively with Linux in mind.

You must run the program as root, probably using `sudo`. You can also grant the executable the `CAP_NET_RAW` capability if you would prefer.

To compile, just run `go build`.
After compilation, you can run the resulting binary with `sudo`.

There are two arguments available:
- `-v` will run the program in verbose mode, causing a LOT of debug prints about the connection management and reliable data transport. This was immensely useful during development, and may be useful to understand how the systems work together.
- `--interface=eth0` will force the program to run on the `eth0` network interface, in the event that the automatic interface selection chooses the wrong interface.

## Key Code
The bulk of the code is in the `networking` package. Key files include:
- `connection.go` - This is where all of the connection management and reliable data transport code lives.
- `ethernet_frame.go` - This is where I handle the raw Ethernet frames, both for sending and receiving.
- `chess_protocol.go` - This is where the Chess packets live. Due to the layering, these same packets could function over a regular TCP socket.
- `layering.go` - This is where the raw frame parsing happens, and each layer is peeled apart and handled individually.