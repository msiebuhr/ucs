Unity Cache Server
==================

.. in Go

Installation from source
------------------------

    go get -u github.com/msiebuhr/ucs/cmd/ucs
    ucs

This will listen for cache-requests on TCP port 8126 and start a small
admin web-server on http://localhost:9126 (currently only servers Promehteus
metrics on /metrics).

Full usage options are shown with `ucs -h`. Note that options can be passed as
environment variables, making the following examples equivalent:

    ucs -quota 10GB
    ucs --quota 10GB
    QUOTA=10GB ucs


As it is generally recommended to [use a cache per major Unity Release and
project](https://github.com/Unity-Technologies/unity-cache-server/issues/50#issuecomment-413854421),
the server supports *namespaces*. This is done by using multiple `-port=NUM` or
`-port=NAME:NUM` arguments:

    ucs -port=8126 -port=name:8127

They will use the same cache, but garbage-collected as one (so old projects'
data will all but vanish and new ones will get lots of space).


Load testing
------------

There's also a quick-and-dirty loadtest utility, `ucs-bender`:

    go get -u github.com/msiebuhr/ucs/cmd/ucs-bender
    ucs-bender # Will run against localhost


Related
-------

 * The "official" [Node.js cache server](https://github.com/Unity-Technologies/unity-cache-server)
 * [Blog about 6.0 development](https://blogs.unity3d.com/2018/03/20/cache-server-6-0-release-and-retrospective-optimizing-import/)
 * [Unity Documentation on Cache Servers](https://docs.unity3d.com/Manual/CacheServer.html)
 * [Unofficial C# Implementation](https://github.com/Avatarchik/UnityCachePlusPlus)

Miscellaneous
-------------

 * Icon by [Elizabeth Arostegui ](https://www.iconfinder.com/icons/998676/challenge_game_puzzle_rubik_icon)
 * MIT-Licensed
