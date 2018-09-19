# Chik

_Chik is the sound my child makes when he runs touching everything around the house_

[![Build Status](https://travis-ci.org/GoChik/chik.svg?branch=master)](https://travis-ci.org/GoChik/chik)

Client and server applications that are part of a simple network of remote enabled devices.    
Each client has an unique identifier, it connects to the server that is acting as a bridge between 
the mobile application that is sending the command and the client that receives the command.  

## Client

The client is able to use GPIO to accomplish different kind of commands. Currently client is tested
on two devices:
 - [RaspberryPI](https://www.raspberrypi.org)
 - [Carambola1](http://www.8devices.com/products/carambola)

Client is configured using the `client.conf` json config file that can be stored either in the same folder
of the executable or inside `/etc/chik/`. 
Configuration file contains two main parameters:
 - server: domain name or ip address and port of the PC where the server instance is running
 - identity: uuid for the client (this field is randomly generated if empty)
 - Depending on the run configuration it may contain additionals sections for gpios and to store saved timers. 

When running the application for the first time the configuration gets automatically created and populated with default values.

## Server

Clients are connecting to the server using a persistent TCP connection TLS encrypted.  
One server can handle multiple clients.  
Server is configured by `server.conf` config file. It can be stored either aside the server executable
or inside `/etc/chik/` directory.  
Server configuration contains following parameters:
 - identity: uuid of the server
 - connection.port: the port the server listens to
 - connection.public_key_path: public key path including filename
 - connection.private_key_path: private key path including filename

When running the application for the first time the configuration gets automatically created and populated with default values and a randomly generated uuid.

## Mobile application

[IoSomePhones](https://github.com/rferrazz/iosomephones)
