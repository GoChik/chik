# Chik

_Chik is the sound my child makes when he runs touching everything around the house_

[![Tag](https://img.shields.io/github/tag/gochik/chik.svg)](https://github.com/gochik/chik/tags)
[![Build Status](https://travis-ci.org/GoChik/chik.svg?branch=master)](https://travis-ci.org/GoChik/chik)
[![Documentation](https://godoc.org/github.com/gochik/chik?status.svg)](https://godoc.org/github.com/gochik/chik)

Framework that allows to create a simple network of remotely connected IOT devices.

The framework allows to compose applications that are instantiating an event loop and a communication bus on top of which a series of [handlers](https://github.com/GoChik/chik/tree/master/handlers) operates. Each handler is providing a different functionality to the application.

Available handlers are:
 - Actor: allows to execute some actions in reaction to a state change or to a series of conditions
 - Heartbeat: sends a periodic heartbeat to check for network connectivity and server availability
 - IO: allows to communicate with various kind of IO devices and protocols
 - Router: allows to route messages between two devices (handler used in server applications)
 - Status: stores a global status comphrensive of every handler state and allows to register remote devices as listener for status changes within the application
 - Sunphase: allows to execute an action at sunrise or sunset (powered by: https://sunrise-sunset.org/api)
 - Timer: allows to execute an action at a certain date and time
 - Version: stores the version of the application and allows OTA updates

Ready made applications:
 - [Embedded client application](https://github.com/GoChik/client)
 - [Server](https://github.com/GoChik/server)
 - [Mobile application](https://github.com/rferrazz/iosomephones)
