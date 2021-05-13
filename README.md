# Chik

_Chik is the sound my child makes when he runs touching everything around the house_

[![Tag](https://img.shields.io/github/tag/gochik/chik.svg)](https://github.com/gochik/chik/tags)
[![Build Status](https://travis-ci.org/GoChik/chik.svg?branch=master)](https://travis-ci.org/GoChik/chik)
[![Documentation](https://godoc.org/github.com/gochik/chik?status.svg)](https://godoc.org/github.com/gochik/chik)

Framework that allows to create a simple network of remotely connected IOT devices.

The framework allows to compose applications that are instantiating an event loop and a communication bus on top of which a series of [handlers](https://github.com/GoChik/chik/tree/master/handlers) operates. Each handler is providing a different functionality to the application.

The framework includes an API to get a TLS encrypted network setup with automatic cert renewal using Smallstep CA APIs (see client and server implementations)

Available handlers are:
 - Actor: allows to execute some actions in reaction to a state change or to a series of conditions
 - Heartbeat: sends a periodic heartbeat to check for network connectivity and server availability
 - IO: allows to communicate with various kind of IO devices and protocols (modbus, GPIO, 1wire and pure software devices)
 - Router: allows to route messages between two devices (handler used in the relay server app)
 - Status: stores a global status comphrensive of every handler state and allows to register remote devices as listener for status changes within the application
 - Datetime: allows to execute an action at a certain date and time or at sunrise/sunset
 - Version: stores the version of the application
 - Telegram: allows to send telegram messages in reaction to a state change
 - Heating: manages zone based heating systems allowing to group small zones together

Ready made applications:
 - [Client](https://github.com/GoChik/client)
 - [Relay Server](https://github.com/GoChik/server)
 - Mobile application
