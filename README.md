# typemapper

Convention based code generator to simplify mapping between similar structs in Go. 

Copying data between similar, but slightly different, structs is error-prone, especially as the size of struct increases. `typemapper` simplifies the process using common conventions to automatically generate mapping functions and helps ensure no fields are left out the copying process but generating unit test files that fail when one is missed.

## How it Works

See the [Tutorial](TUTORIAL.md).

## TODO

* [ ] Sync tutorial example files with markdown doc