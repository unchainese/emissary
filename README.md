# unchain

"Unchainese" – a blend of "unchain" and "Chinese" – represents breaking free from internet restrictions and embracing true digital freedom.

Unchain is design in Go to be a simple and easy to use proxy server 
that can be used to bypass network restrictions,censorship and surveillance.


Unchain accepts traffic from the client eg: v2rayN,v2rayA,v2rayNG,shadowRocket etc.
Processes the traffic and forwards it to the destination server.

## Unchain Architecture

Unchain server uses a simple architecture that is VLESS over WebSocket (WS) + TLS.


```
             V2rayN,V2rayA,Clash or ShadowRocket                          
                 +------------------+
                 |   VLESS Client   |
                 |   +-----------+  |
                 |   | TLS Layer  |  |
                 |   +-----------+  |
                 |   | WebSocket  |  |
                 |   +-----------+  |
                 +--------|---------+
                          |
                          | Encrypted VLESS Traffic (wss://)
                          |
           +--------------------------------------+
           |         Internet (TLS Secured)      |
           +--------------------------------------+
                          |
                          |
        +-----------------------------------+
        |        Reverse Proxy Server       |
        | (e.g., Nginx or Cloudflare)       |
        |                                   |
        |   +---------------------------+   |
        |   | HTTPS/TLS Termination     |   |
        |   +---------------------------+   |
        |   | WebSocket Proxy (wss://)  |   |
        |   +---------------------------+   |
        |     Forward to VLESS Server     |
        +------------------|----------------+
                           |
           +--------------------------------+
           |     Unchain VLESS Server       |
           |                                |
           |   +------------------------+   |
           |   | WebSocket Handler      |   |
           |   +------------------------+   |
           |   | VLESS Core Processing  |   |
           |   +------------------------+   |
           |                                |
           |   Forward Traffic to Target    |
           +------------------|-------------+
                              |
                     +-----------------+
                     | Target Server   |
                     | or Destination  |
                     +-----------------+

```