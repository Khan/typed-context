This is the sample code for a [post on the Khan Academy Engineering Blog
describing how we use statically typed context](https://blog.khanacademy.org/statically-typed-context-in-go/).

-----

Khan/typed-context is not accepting contributions. We’re releasing the code for
others to refer to and learn from, but we are not open to pull requests or
issues at this time.

Khan Academy is a non-profit organization with a mission to provide a free,
world-class education to anyone, anywhere. You can help us in that mission by
[donating](https://khanacademy.org/donate) or looking at
[career opportunities](https://khanacademy.org/careers).

-----

There are 5 examples, each of which is described in the blog post.  In each
case the file `thing.go` contains a function `DoTheThing` which does the same
things.  They vary in how they access global & request specific elements.

1. Globals
2. Parameters
3. Context, with unsafe casting
4. Context but safely
5. statically typed context

The blog post explains the pros and cons, and explains why we use statically
typed contexts at Khan Academy.

There's also a `main` that calls `DoTheThing`.  Comparing the differences
between the examples allows you to see how easy it is to write functions and
call functions using the various techniques.

Each example also has a file `mock.go` which has stub implementations of the
functions called in `DoTheThing`.  The last example also has `contexts.go`
which defines the interface types used in the example.

Finally there's a linter that can detect when a function declaration requires an
interface type that is not used in the code.  We use this linter to ensure every
function declares the minimal interface possible, which ensures the interface
list is an accurate list of dependencies.  If you run it against example 5 it
passes.  In `mocks.go` there are lines like `_ = ctx.Request()` that exist to
satisfy the linter.  If those lines are removed the linter will fail.

We use statically typed contexts within Khan Academy.  If you like the idea and
are excited to use them at work, [we're hiring](https://www.khanacademy.org/careers).

