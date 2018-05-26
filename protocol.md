# Server Protocol

Numbers are sent back end forth in hex-encoding. Eg. the version is sent as
`000000fe` over the wire (and *not* the binary `000\u00fe`).

The commentary below is from reading package dumps and looking at the
unit-tests on the canonical server implementation.

## Version check

```
client --- (version <uint32>) --> server	  (using version)
client <-- (version <uint32>) --- server	  (echo version if supported or 0)
```

The server reads for eight bytes of data from the first package received. If
the package contain less than eight bytes, only those are used. Only exception
if getting a one-byte package, in which case it should wait for the next
package in order go get at least two bytes worth of data.

The only accepted client version is 254 (`0xfe` / `0x000000fe`), to which the
server answers `0x000000fe`. In all other cases the server replies `0x00000000`
and closes the connection.

## Request cached item
```
client --- 'ga' (id <128bit GUID><128bit HASH>) --> server
client <-- '+a' (size <uint64>) (id <128bit GUID><128bit HASH>) + size bytes --- server (found in cache)
client <-- '-a' (id <128bit GUID><128bit HASH>) --- server (not found in cache)

client --- 'gi' (id <128bit GUID><128bit HASH>) --> server
client <-- '+i' (size <uint64>) (id <128bit GUID><128bit HASH>) + size bytes --- server (found in cache)
client <-- '-i' (id <128bit GUID><128bit HASH>) --- server (not found in cache)

client --- 'gr' (id <128bit GUID><128bit HASH>) --> server
client <-- '+r' (size <uint64>) (id <128bit GUID><128bit HASH>) + size bytes --- server	(found in cache)
client <-- '-r' (id <128bit GUID><128bit HASH>) --- server (not found in cache)
```
## Start transaction
```
client --- 'ts' (id <128bit GUID><128bit HASH>) --> server
```

## Put cached item
```
client --- 'pa' (size <uint64>) + size bytes --> server
client --- 'pi' (size <uint64>) + size bytes --> server
client --- 'pr' (size <uint64>) + size bytes --> server
```

## End transaction (i.e. rename targets to their final names)
```
client --- 'te' --> server
```
## Quit
```
client --- 'q' --> server
```