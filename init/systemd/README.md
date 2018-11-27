System D install
================

First, create the user and place systemd + config files in the expected places

    adduser -s unitycacheserver
    cp ucs.service /etc/systemd/services/
    cp ucs.conf /etc/ucs.conf
    sudoedit /etc/ucs.conf
    systemctl enable ucs

A built binary is expected in `/usr/local/bin/ucs`, but can really be put
anywhere one feels like (as long as `ucs.service` is edited accordingly).

    go build ../../cmd/ucs -o ucs
    mv ucs /usr/local/bin/

Finally, make the cache directory in `/var/cache/ucs`

    sudo -u unitycacheserver /var/cache/ucs

UCS can then be started and poked at

    systemctl start ucs
    open http://localhost:9126 # Web interface
    echo 000000fe | nc localhost 8126 # Do a quick handshake
    000000fe%

Prometheus metrics will by default be available at
http://localhost:9126/metrics
