# IOSomething

_Internet of something_

Client and server applications that are part of a simple network of remote enabled devices.    
Each client has an unique identifier, it connects to the server that is acting as a bridge between 
the mobile application that is sending the command and the client that receives the command.  

## Client

The client is able to use GPIO to accomplish different kind of commands. Currently client is tested
on two devices:
 - [RaspberryPI](https://www.raspberrypi.org)
 - [Carambola1](http://www.8devices.com/products/carambola)

Client is configured using the `client.json` config file that can be stored either in the same folder
of the executable or inside `/etc/iosomething/`. 
Configuration file contains two parameters:
 - Server: domain name or ip address and port of the PC where the server instance is running
 - Identity: uuid for the client (this field is randomly generated if empty)

Following an example of a valid configuration file:
```
{
  "Server": "mydomain.com:6767",
  "Identity": "dc9c465c-c8cc-11c6-884c-6c40089ac3c6"
}
```

## Server

Server sends tls encrypted messages. One server can handle multiple clients.  
Server is configured by `server.json` config file. It can be stored either aside the server executable
or inside `/etc/iosomething/` directory.  
Server configuration contains following parameters:
 - Port: port to use to communicate
 - PubKeyPath: public key path with filename
 - PrivKeyPath: private key path with filename

 Example:
 ```
 {
    "port": 6767,
    "PubKeyPath": "/etc/iosomething/cert.pem",
    "PrivKeyPath": "/etc/iosomething/key.pem"
}
 ```

## Mobile application

_Wait for it_