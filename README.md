# Yubikey

Messing with yubikeys and the web authentication protocol.

Excellent blog post on the topic: [www.herbie.dev/blog/webauthn-basic-web-client-server/](https://www.herbie.dev/blog/webauthn-basic-web-client-server/).

## Networking

Edit the `/etc/hosts` file on your computer and add the following line:

```
127.0.0.1 yubikey.local
```

Note you will have to edit this file in `sudo` mode to save it.

## TLS Certificates

Generate self-signed certificates as follows:

```
$ openssl genrsa -out tmp/server.key 2048
$ openssl req -new -x509 -sha256 -key tmp/server.key -out tmp/server.crt -days 3650
```

Make sure that the FQDN is yubikey.local or whatever you added for networking above.