
The `var/` directory need to be writable by non-root since our docker image uses the `nonroot` user with id `65532`.

It will hold files like log files, eventually.

Make sure you backup this directory !
