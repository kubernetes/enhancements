# Experimenting with Go behaviors wrt "v2"

The document details an experiment performed to see exactly what Go does with
regards to v2.

TL;DR: It seems OK with no tags in play, but if there are tags it seems to
break down.

<!-- toc -->
- [Setup](#setup)
- [Edits without tags](#edits-without-tags)
- [Introducing v2](#introducing-v2)
- [With tags](#with-tags)
<!-- /toc -->

## Setup

I created a fake-lib repo:

```
thockin-glaptop4 fake-lib 0b906 main /$ cat go.mod
module github.com/thockin/fake-lib

go 1.21.3

thockin-glaptop4 fake-lib 0b906 main /$ cat fake.go
package fakelib

var X = "fake lib v0.0.0"
```

I created a fake-app repo:

```
thockin-glaptop4 fake-app 8efd0 main /$ cat go.mod
module github.com/thockin/fake-app

go 1.21.3

thockin-glaptop4 fake-app 8efd0 main /$ cat app.go
package main

import (
	"fmt"

	fakelib "github.com/thockin/fake-lib"
)

func main() {
	fmt.Println(fakelib.X)
}

thockin-glaptop4 fake-app 8efd0 main /$ go run .
app.go:6:2: no required module provides package github.com/thockin/fake-lib; to add it:
	go get github.com/thockin/fake-lib

thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get github.com/thockin/fake-lib
go: downloading github.com/thockin/fake-lib v0.0.0-20240118195027-0b90652648a0
go: upgraded github.com/thockin/fake-lib v0.0.0-20240118184109-2ef452a3761b => v0.0.0-20240118195027-0b90652648a0

thockin-glaptop4 fake-app 8efd0 main /$ cat go.mod
module github.com/thockin/fake-app

go 1.21.3

require github.com/thockin/fake-lib v0.0.0-20240118195027-0b90652648a0 // indirect

thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v0.0.0
```

## Edits without tags

I changed the lib:

```
commit 8c845166b6b12838779b995919cc3d7f6fedd62c (HEAD -> main)
Good "git" signature for thockin@google.com with ED25519 key SHA256:M2+rk0pn34LtCBIVneq+UkpS+MLUTLylIUjHPq6OGSE
Author: Tim Hockin <thockin@google.com>
Date:   Thu Jan 18 12:00:26 2024 -0800

    v0.0.0 edit 1

diff --git a/fake.go b/fake.go
index 4dd1ccd..273a71e 100644
--- a/fake.go
+++ b/fake.go
@@ -1,3 +1,3 @@
 package fakelib

-var X = "fake lib v0.0.0"
+var X = "fake lib v0.0.0 edit 1"
```

In the app:

```
thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib
go: downloading github.com/thockin/fake-lib v0.0.0-20240118200026-8c845166b6b1
go: upgraded github.com/thockin/fake-lib v0.0.0-20240118195027-0b90652648a0 => v0.0.0-20240118200026-8c845166b6b1

thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v0.0.0 edit 1
```

This is effectively what we do with repos like `k8s.io/gengo` today.

## Introducing v2

I made fake-lib/v2 without any tags:

```
thockin-glaptop4 fake-lib 7f1be main /v2$ git show
commit 7f1be8fbd1ee46c2c84483002f97c5f43730f577 (HEAD -> main)
Good "git" signature for thockin@google.com with ED25519 key SHA256:M2+rk0pn34LtCBIVneq+UkpS+MLUTLylIUjHPq6OGSE
Author: Tim Hockin <thockin@google.com>
Date:   Thu Jan 18 12:03:57 2024 -0800

    v2 untagged

diff --git a/v2/fake.go b/v2/fake.go
new file mode 100644
index 0000000..0601f26
--- /dev/null
+++ b/v2/fake.go
@@ -0,0 +1,3 @@
+package fakelib
+
+var X = "fake lib v2 untagged"
diff --git a/v2/go.mod b/v2/go.mod
new file mode 100644
index 0000000..d6b26fb
--- /dev/null
+++ b/v2/go.mod
@@ -0,0 +1,3 @@
+module github.com/thockin/fake-lib/v2
+
+go 1.21.3
```

In the app, I tried a naive (I knew it won't work) approach:

```
thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib
go: downloading github.com/thockin/fake-lib v0.0.0-20240118200357-7f1be8fbd1ee
go: upgraded github.com/thockin/fake-lib v0.0.0-20240118200026-8c845166b6b1 => v0.0.0-20240118200357-7f1be8fbd1ee

thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v0.0.0 edit 1

thockin-glaptop4 fake-app 8efd0 main /$ sed -i 's|fake-lib|fake-lib/v2|' app.go

thockin-glaptop4 fake-app 8efd0 main /$ go run .
app.go:6:2: no required module provides package github.com/thockin/fake-lib/v2; to add it:
	go get github.com/thockin/fake-lib/v2
```

OK, as expected.  Let's do it right.

```
thockin-glaptop4 fake-app 8efd0 main /$ git co -- .

thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get github.com/thockin/fake-lib/v2
go: downloading github.com/thockin/fake-lib/v2 v2.0.0-20240118200357-7f1be8fbd1ee
go: added github.com/thockin/fake-lib/v2 v2.0.0-20240118200357-7f1be8fbd1ee

thockin-glaptop4 fake-app 8efd0 main /$ sed -i 's|fake-lib|fake-lib/v2|' app.go

thockin-glaptop4 fake-app 8efd0 main /$ go mod tidy

thockin-glaptop4 fake-app 8efd0 main /$ cat go.mod
module github.com/thockin/fake-app

go 1.21.3

require github.com/thockin/fake-lib/v2 v2.0.0-20240118200357-7f1be8fbd1ee

thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v2 untagged
```

OK so far - back to the lib, make some edits:

```
thockin-glaptop4 fake-lib 58927 main /v2$ git show
commit 589276a8e3e3909948b34ffeeca402b6f4691102 (HEAD -> main)
Good "git" signature for thockin@google.com with ED25519 key SHA256:M2+rk0pn34LtCBIVneq+UkpS+MLUTLylIUjHPq6OGSE
Author: Tim Hockin <thockin@google.com>
Date:   Thu Jan 18 12:10:10 2024 -0800

    v2 untagged, edit 1

diff --git a/v2/fake.go b/v2/fake.go
index 0601f26..bcaf878 100644
--- a/v2/fake.go
+++ b/v2/fake.go
@@ -1,3 +1,3 @@
 package fakelib

-var X = "fake lib v2 untagged"
+var X = "fake lib v2 untagged, edit 1"
```

And in the app:

```
thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2
go: downloading github.com/thockin/fake-lib/v2 v2.0.0-20240118201010-589276a8e3e3
go: downloading github.com/thockin/fake-lib v0.0.0-20240118201010-589276a8e3e3
go: upgraded github.com/thockin/fake-lib/v2 v2.0.0-20240118200357-7f1be8fbd1ee => v2.0.0-20240118201010-589276a8e3e3

thockin-glaptop4 fake-app 8efd0 main /$ go mod tidy

thockin-glaptop4 fake-app 8efd0 main /$ cat go.mod
module github.com/thockin/fake-app

go 1.21.3

require github.com/thockin/fake-lib/v2 v2.0.0-20240118201010-589276a8e3e3

thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v2 untagged, edit 1
```

That seems to work.  For good measure, one more time:

```
thockin-glaptop4 fake-lib 64082 main /v2$ git show
commit 640822f7fd61faed90ac05005e027b2a875787b9 (HEAD -> main, origin/main, origin/HEAD)
Good "git" signature for thockin@google.com with ED25519 key SHA256:M2+rk0pn34LtCBIVneq+UkpS+MLUTLylIUjHPq6OGSE
Author: Tim Hockin <thockin@google.com>
Date:   Thu Jan 18 12:13:40 2024 -0800

    v2 untagged, edit 2

diff --git a/v2/fake.go b/v2/fake.go
index bcaf878..27d791e 100644
--- a/v2/fake.go
+++ b/v2/fake.go
@@ -1,3 +1,3 @@
 package fakelib

-var X = "fake lib v2 untagged, edit 1"
+var X = "fake lib v2 untagged, edit 2"
```

```
thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2
go: downloading github.com/thockin/fake-lib/v2 v2.0.0-20240118201340-640822f7fd61
go: downloading github.com/thockin/fake-lib v0.0.0-20240118201340-640822f7fd61
go: upgraded github.com/thockin/fake-lib/v2 v2.0.0-20240118201010-589276a8e3e3 => v2.0.0-20240118201340-640822f7fd61

thockin-glaptop4 fake-app 8efd0 main /$ go mod tidy

thockin-glaptop4 fake-app 8efd0 main /$ cat go.mod
module github.com/thockin/fake-app

go 1.21.3

require github.com/thockin/fake-lib/v2 v2.0.0-20240118201340-640822f7fd61

thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v2 untagged, edit 2
thockin-glaptop4 fake-app 8efd0 main /$
```

## With tags

It all seems to work without any tags.  Let's mess it up.

lib:

```
thockin-glaptop4 fake-lib 0e7a2 main /v2$ git show
commit 0e7a25e511f2b7893ce39aefa1c3dc83592bd734 (HEAD -> main)
Good "git" signature for thockin@google.com with ED25519 key SHA256:M2+rk0pn34LtCBIVneq+UkpS+MLUTLylIUjHPq6OGSE
Author: Tim Hockin <thockin@google.com>
Date:   Thu Jan 18 12:16:52 2024 -0800

    v2.0.0 tagged

diff --git a/v2/fake.go b/v2/fake.go
index 27d791e..bdf0e8a 100644
--- a/v2/fake.go
+++ b/v2/fake.go
@@ -1,3 +1,3 @@
 package fakelib

-var X = "fake lib v2 untagged, edit 2"
+var X = "fake lib v2.0.0 tagged"

thockin-glaptop4 fake-lib 0e7a2 main /v2$ git tag v2.0.0

thockin-glaptop4 fake-lib 0e7a2 main /v2$ git push --tags
```

app:

```
thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2

thockin-glaptop4 fake-app 8efd0 main /$ # That had no effect

thockin-glaptop4 fake-app 8efd0 main /$ cat go.mod
module github.com/thockin/fake-app

go 1.21.3

require github.com/thockin/fake-lib/v2 v2.0.0-20240118201340-640822f7fd61

thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2@latest
go: downloading github.com/thockin/fake-lib/v2 v2.0.0
go: upgraded github.com/thockin/fake-lib/v2 v2.0.0-20240118201340-640822f7fd61 => v2.0.0

thockin-glaptop4 fake-app 8efd0 main /$ go mod tidy

thockin-glaptop4 fake-app 8efd0 main /$ cat go.mod
module github.com/thockin/fake-app

go 1.21.3

require github.com/thockin/fake-lib/v2 v2.0.0

thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v2.0.0 tagged
```

And let's bump it without tags:

```
thockin-glaptop4 fake-lib 203cf main /v2$ git show
commit 203cfc208120f01d03ce671dc65a087c91d79c3c (HEAD -> main, origin/main, origin/HEAD)
Good "git" signature for thockin@google.com with ED25519 key SHA256:M2+rk0pn34LtCBIVneq+UkpS+MLUTLylIUjHPq6OGSE
Author: Tim Hockin <thockin@google.com>
Date:   Thu Jan 18 12:20:15 2024 -0800

    v2.0.0 tagged, edit 1

diff --git a/v2/fake.go b/v2/fake.go
index bdf0e8a..7d33402 100644
--- a/v2/fake.go
+++ b/v2/fake.go
@@ -1,3 +1,3 @@
 package fakelib

-var X = "fake lib v2.0.0 tagged"
+var X = "fake lib v2.0.0 tagged, edit 1"
```

```
thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2
go: downloading github.com/thockin/fake-lib v0.0.0-20240118202015-203cfc208120

thockin-glaptop4 fake-app 8efd0 main /$  # Whoah, not what I expected: v0.0.0 ???

thockin-glaptop4 fake-app 8efd0 main /$ cat go.mod
module github.com/thockin/fake-app

go 1.21.3

require github.com/thockin/fake-lib/v2 v2.0.0
thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v2.0.0 tagged
```

So the tag's existence broke `go get -u`.  Let's try some other things:

```
thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2@latest

thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2@master
go: github.com/thockin/fake-lib/v2@master: invalid version: unknown revision master

thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2@main
go: downloading github.com/thockin/fake-lib/v2 v2.0.1-0.20240118202015-203cfc208120
go: upgraded github.com/thockin/fake-lib/v2 v2.0.0 => v2.0.1-0.20240118202015-203cfc208120

thockin-glaptop4 fake-app 8efd0 main /$ go mod tidy

thockin-glaptop4 fake-app 8efd0 main /$ cat go.mod
module github.com/thockin/fake-app

go 1.21.3

require github.com/thockin/fake-lib/v2 v2.0.1-0.20240118202015-203cfc208120
thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v2.0.0 tagged, edit 1
```

OK, so naming the branch worked.  That's unpleasant because it is whatever the
lib wants.  I pushed a lib "edit 2"

```
 $ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2
go: downloading github.com/thockin/fake-lib/v2 v2.0.1-0.20240118202458-f889dc37dc89
go: downloading github.com/thockin/fake-lib v0.0.0-20240118202458-f889dc37dc89
go: upgraded github.com/thockin/fake-lib/v2 v2.0.1-0.20240118202015-203cfc208120 => v2.0.1-0.20240118202458-f889dc37dc89

thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v2.0.0 tagged, edit 2
```

Well that is even WEIRDER.  I pushed an "edit 3"

```
$ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2@HEAD
go: downloading github.com/thockin/fake-lib/v2 v2.0.1-0.20240118202641-9ce247c3b4f4
go: downloading github.com/thockin/fake-lib v0.0.0-20240118202641-9ce247c3b4f4
go: upgraded github.com/thockin/fake-lib/v2 v2.0.1-0.20240118202458-f889dc37dc89 => v2.0.1-0.20240118202641-9ce247c3b4f4

thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v2.0.0 tagged, edit 3
```

That works, too.  Let's make another tag.

```
$ git show
commit 0056b80c4b59c5499185348176f56a75c763d2f6 (HEAD -> main)
Good "git" signature for thockin@google.com with ED25519 key SHA256:M2+rk0pn34LtCBIVneq+UkpS+MLUTLylIUjHPq6OGSE
Author: Tim Hockin <thockin@google.com>
Date:   Thu Jan 18 12:28:09 2024 -0800

    v2.1.0 tagged

diff --git a/v2/fake.go b/v2/fake.go
index fccb833..81a9c0c 100644
--- a/v2/fake.go
+++ b/v2/fake.go
@@ -1,3 +1,3 @@
 package fakelib

-var X = "fake lib v2.0.0 tagged, edit 3"
+var X = "fake lib v2.1.0 tagged"
```

```
thockin-glaptop4 fake-app 8efd0 main /$ GOPROXY=direct go get -u github.com/thockin/fake-lib/v2
go: downloading github.com/thockin/fake-lib/v2 v2.1.0
go: upgraded github.com/thockin/fake-lib/v2 v2.0.1-0.20240118202641-9ce247c3b4f4 => v2.1.0

thockin-glaptop4 fake-app 8efd0 main /$ go run .
fake lib v2.1.0 tagged
```

So, final anwer: It works as we want without tags.  Once tags are in play, we
have to keep using tags.  The v0.0.0 behavior seems like a bug, but maybe not
worth hunting.
