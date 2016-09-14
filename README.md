
# NDN-Client
Named Data Networking (NDN) is a promising paradigm for the future Internet architecture which opens up new possibilities for the data exchange among routers. In order to learn NDN principles, a simpler NDN protocol has been developed in a mobile environment by the means of different
boards. This client is able to interact with the Arduino library (the server counterpart) **NDNOverUDP**.

## Usage
It has been developed to be used as a swiss-knife for the
NDN protocol, it has multiple features which will be described
below.
When invoked from the command line it requires one manda-
tory argument, which is the interest name:

`$ ndn−client [OPTION]... INTEREST`

## Inner workings
Given the interest's name the client proceeds to broadcast the
interest packet as a UDP datagram on the local network on
the port 8888 (which has been chosen as the port of reference
for the NDN protocol), then it listens on the same port for
any incoming Data packet and prints on standard output the
content of the just arrived packet.
As shown by the usage string above the ndn-client has many
other features which can be used through command line
options, the following is a short description of them:

* **-sd**, send a Data packet instead of a Interest one;
* **-c** ”string”, content of the Data packet to be sent;
* **-dd**, Print dump of the received Data packet;
* **-di**, Print dump of the sent Interest packet;
* **-x**, Print a hex dump;
* **-nl**, Do not wait for a response Data packet;
* **-gw**, Instead of broadcasting the packet, use a NDN Gateway;
* **--intel**, Supply this option if using a network composed of Intel Galileo;

### Why?
If trying the NDNOverUDP library with some Arduinos or any compatible board this is useful to debug, retrieve content or take a look at an example of a simple NDN implementation over UDP.

------------------------------------------------------------
Copyright (C) **Antonio Cardace** and **Davide Aguiari** 2016, antonio@cardace.it, gorghino@gmail.com
