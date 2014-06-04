Admin example
=============

This is a simple example application using the admin panel.

The example uses Beego's ORM and net/http. Note that the admin package does not depend on any specific ORM, as long as a struct -> sql mapping exists.

Run it
------

    go build
    ./example orm syncdb
    ./example

And visit http://localhost:8000/admin/