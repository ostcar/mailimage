# Mailimage

An image block, where the images are posted via mail.


## Build

To build the binary, clone the repo and call:

```
$ go generate
$ go build
```

## Install


To run mailimage, you have to install [redis](https://redis.io/).

Also you have to install and configure a mail transfer agend, that can redirect
an incomming mail to a programm. You can use [Postfix](http://www.postfix.org/)
for example.


### Postfix

To configure postfix you have to put a line like

```
mailimage: |"/path/to/mailimage insert"
```

To the file ```/etc/aliases``` an run ```newaliases```.

After this you can use the file ```/etc/postfix/virtual``` to send mails to the
fictive user ```mailimage```.


### Webserver

To start the mailimage webserver call

```
mailimage serve
```

You can use an webserver like [nginx](https://nginx.org/en/) as a proxy. For example
with this configuration:

```
server {
  server_name  mailimage.oshahn.de;

      include include/plain;

      root /srv/sftp/openslides/files;

    location / {
        proxy_pass http://localhost:5000;
    }

}
```
