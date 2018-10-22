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
