# Mailimage

An image block, where the images are posted via mail.


## Install

Run

```
$ go get github.com/ostcar/mailimage
```

to install the program.

## Dependencies


To run mailimage, you have to install [redis](https://redis.io/).

Also you have to install and configure a mail transfer agend, that can redirect
an incomming mail to a programm. You can use [Postfix](http://www.postfix.org/)
for example.


### Postfix

To configure postfix you have to put a line like

```
mailimage: |"/path/to/mailimage insert"
```

To the file ```/etc/aliases``` and run ```newaliases```.

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
  server_name  mailimage.baarfood.de;

    include include/plain;

    location / {
        proxy_pass http://localhost:5000;
    }

}
```
