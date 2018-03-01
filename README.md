# transport #

**An extendable toolkit for improving the standard library HTTP client.**

## Usage ##

### Creating A Transport ###

The standard library `http.Transport` implementation is largely sufficient for
for most uses of `http.Client`. However, the `http.DefaultTransport` used by
the client is not configured for production use. Several of the timeout values,
TLS handshake for example, are set in the seconds. Likewise, the dial methods
used to establish TCP connections are allowed upwards of thirty seconds before
they fail. Recreating the `http.DefaultTransport` settings before overwriting
the value necessary for a production environment can be tedious. To help, this
package offers an alternative constructor for `http.Transport` that leverages
functional arguments to make it easier to configure. For example:

```golang
var client = &http.Client{
  Transport: transport.New(
    transport.OptionDefaultTransport,
    transport.OptionMaxResponseHeaderBytes(4096),
    transport.OptionDisableCompression(true),
  ),
}
```

Additionally, the same options may be used to create a factory function which
is able to produce any number of transports with the same configuration:

```golang
var factory = transport.NewFactory(
  transport.OptionDefaultTransport,
  transport.OptionMaxResponseHeaderBytes(4096),
  transport.OptionDisableCompression(true),
)
var client1 = &http.Client{
  Transport: factory(),
}
var client2 = &http.Client{
  Transport: factory(),
}
```

### Decorators ###

In addition to providing the transport constructor, this package provides a
handful of tools that make operating the `http.Transport` a little easier for
complex use cases. Each of these additions comes in the form of a wrapper
around the transport in a way that is seamless to the `http.Client` and any
code that uses the `http.Client`.

#### Retry ####

One of the most common needs for network based code is to retry on intermittent,
or transient, errors. To help with this use case, this package provides a retry
decorator that can be configured to retry on a number of conditions within a
number of limits without adding more complexity to code using the `http.Client`.

```golang
var retryDecorator = transport.NewRetrier(
  transport.RetrierOptionResponseCode(http.StatusInternalServerError),
  transport.RetrierOptionTimeout(100*time.Millisecond),
  transport.RetrierOptionLimit(3),
  transport.RetrierOptionDelay(50*time.Millisecond),
  transport.RetrierOptionDelayJitter(30*time.Millisecond),
)
var t = transport.New(
  transport.OptionDefaultTransport,
  transport.OptionMaxResponseHeaderBytes(4096),
  transport.OptionDisableCompression(true),
)
var client = &http.Client{
  Transport: retryDecorator(t),
}
```

The above snippet adds retry logic that:

  -   Retries up to three times before failing
  -   Adds a jittered delay between each retry of 20ms and 80ms
  -   Retries automatically if the response code is 500
  -   Cancels an active request and retries if it takes longer than 100ms

#### Headers ####

Another common need is to inject headers automatically into outgoing requests
so that application code doesn't have to be aware of elements like
authentication or tracing. For these cases, this package provides a header
injection decorator:

```golang
var headerDecorator = transport.NewHeader(
  func(*http.Request) (string, string) {
    return "Bearer", os.Getenv("SECURE_TOKEN")
  }
)
var t = transport.New(
  transport.OptionDefaultTransport,
  transport.OptionMaxResponseHeaderBytes(4096),
  transport.OptionDisableCompression(true),
)
var client = &http.Client{
  Transport: headerDecorator(t),
}
```

The above snippet configures the transport to automatically inject an auth
token into the headers on each request. The constructor takes any function
matching the signature shown above to allow for any level of complexity in
selecting the header name and value.

#### Decorator Chains ####

Most use cases require more than one decorator. To help, this package provides
a decorator chain implementation that can be used to collect a series of
decorator behaviours and have them applied in a specific order to any given
transport:

```golang
var retryDecorator = transport.NewRetrier(
  transport.RetrierOptionResponseCode(http.StatusInternalServerError),
  transport.RetrierOptionTimeout(100*time.Millisecond),
  transport.RetrierOptionLimit(3),
  transport.RetrierOptionDelay(50*time.Millisecond),
  transport.RetrierOptionDelayJitter(30*time.Millisecond),
)
var headerDecorator = transport.NewHeader(
  func(*http.Request) (string, string) {
    return "Bearer", os.Getenv("SECURE_TOKEN")
  }
)
var chain = transport.Chain{
  retryDecorator
  headerDecorator,
}
var t = transport.New(
  transport.OptionDefaultTransport,
  transport.OptionMaxResponseHeaderBytes(4096),
  transport.OptionDisableCompression(true),
)
var client = &http.Client{
  Transport: chain.Apply(t),
}
```

The decorators will be applied in the reverse order they are given. Another
way to think of this is that the request will pass through the decorators
in the same order they are given. For example, a chain containing middleware
`A`, `B`, `C`, and `D` will be applied like:

```
A(B(C(D(TRANSPORT))))
```

### Transport Extensions ###

Decorators are a powerful pattern and a great deal of complexity can be isolated
by using them. However, there are still some aspects of the core
`http.Transport` behaviour that can be harmful in production if not altered.
This package provides some modifications of the standard behaviour to account
for these cases.

#### Recycle Transport ####

The default settings of the `http.Transport` include enabling the connection
pool. Having a connection pool can be a highly effective optimisation by
allowing the cost of performing DNS lookups, acquiring a TCP connection, and
performing TLS handshakes to be amortised over a potentially large number of
outgoing requests.

One of the major deficiencies of the built-in connection pool is that there are
no limits on connection lifetime. Granted, there are limits on *connection idle*
time but these limits only apply when a connection goes unused. A higher scale
service may see that connections never go idle. If a service is using DNS in
order to connect to an endpoint then it can miss a change in the DNS records
because it does not generate new connections frequently enough. To help with
this issue, the package provides a transport modifier that can reset the entire
connection pool on certain triggers.

```golang
var retryDecorator = transport.NewRetrier(
  transport.RetrierOptionResponseCode(http.StatusInternalServerError),
  transport.RetrierOptionTimeout(100*time.Millisecond),
  transport.RetrierOptionLimit(3),
  transport.RetrierOptionDelay(50*time.Millisecond),
  transport.RetrierOptionDelayJitter(30*time.Millisecond),
)
var headerDecorator = transport.NewHeader(
  func(*http.Request) (string, string) {
    return "Bearer", os.Getenv("SECURE_TOKEN")
  }
)
var chain = transport.Chain{
  retryDecorator
  headerDecorator,
}
var factory = transport.NewFactory(
  transport.OptionDefaultTransport,
  transport.OptionMaxResponseHeaderBytes(4096),
  transport.OptionDisableCompression(true),
)
var finalTransport = transport.NewRecycler(
  chain.ApplyFactory(factory),
  transport.RecycleOptionTTL(5*time.Minute),
  transport.RecycleOptionTTLJitter(1*time.Minute),
)
var client = &http.Client{Transport: finalTransport}
```

Building on the decorator examples, in this snipper we construct a new transport
factory that is bound to a set of decorators that add functionality. Then we
wrap the factory in a recycler that is configured to refresh the connection
pool every five minutes with a randomised jitter within +/- one minute.

*Note: There is currently no reliable way by which per-connection lifetime
limits can be added. We are limited to managing the entire pool.*

#### Rotating Transport ####

The internal connection management strategies of the standard library HTTP/1 and
HTTP/2 transports are quite different. The HTTP/1 transport must use a single
connection per-request. If it is attempting to make a new request and there are
no idle connections in the pool then it will make a new connection for that
request. The HTTP/2 transport, however, re-uses a single TCP connection for
all requests.

When communicating with an HTTP server that supports HTTP/2 the `http.Transport`
automatically creates an HTTP/2 transport internally and re-routes all requests
through it. Oftentimes, this is a great optimisation that we get for free.
However, there are some edge cases around using a single connection for all
outgoing requests. One of the larger edge cases is related to increased latency
when experiencing packet loss. Several folks
[have written](https://www.twilio.com/blog/2017/10/http2-issues.html) about
[this problem](https://hpbn.co/http2/) if you're looking for more details.

As a tool for managing the impact of this particular problem, this package
provides a transport modifier that is capable of creating and maintaining
multiple connection pools for a single destination to ensure that requests
are spread evenly over multiple TCP connection even when in HTTP/2 mode:

```golang
var retryDecorator = transport.NewRetrier(
  transport.RetrierOptionResponseCode(http.StatusInternalServerError),
  transport.RetrierOptionTimeout(100*time.Millisecond),
  transport.RetrierOptionLimit(3),
  transport.RetrierOptionDelay(50*time.Millisecond),
  transport.RetrierOptionDelayJitter(30*time.Millisecond),
)
var headerDecorator = transport.NewHeader(
  func(*http.Request) (string, string) {
    return "Bearer", os.Getenv("SECURE_TOKEN")
  }
)
var chain = transport.Chain{
  retryDecorator
  headerDecorator,
}
var factory = transport.NewFactory(
  transport.OptionDefaultTransport,
  transport.OptionMaxResponseHeaderBytes(4096),
  transport.OptionDisableCompression(true),
)
var recycleFactory = transport.NewRecyclerFactory(
  chain.ApplyFactory(factory),
  transport.RecycleOptionTTL(5*time.Minute),
  transport.RecycleOptionTTLJitter(1*time.Minute),
)
var finalTransport = transport.NewRotator(
  recycleFactory,
  transport.RotatorOptionInstances(5),
)
var client = &http.Client{Transport: finalTransport}
```

The above is meant to illustrate two things. The first is the configuration of
a rotator modification that ensures there are always at least five TCP
connections in use for each HTTP/2 endpoint rather than one. The other is how
the tools of this package can be composited with each other. The example above
configures each of the five connection pools to recycle every four to five
minutes just like the previous example that focused on the recycler.

Underneath, this option will cause the service to actually create five,
individual transports using whichever factory function is given to it. The
requests made through the client will be spread across the transports using a
round-robin policy.

It is important to note that using this option for HTTP/1 connections may make
the connection pooling *worse* because the connection management is so
different. **It is only recommended to use this option with HTTP/2 connections.**

## Contributing ##

### License ###

This project is licensed under Apache 2.0. See LICENSE.txt for details.

### Contributing Agreement ###

Atlassian requires signing a contributor's agreement before we can accept a
patch. If you are an individual you can fill out the
[individual CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=3f94fbdc-2fbe-46ac-b14c-5d152700ae5d).
If you are contributing on behalf of your company then please fill out the
[corporate CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=e1c17c66-ca4d-4aab-a953-2c231af4a20b).
