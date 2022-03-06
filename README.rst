============================================
``emt`` -- Error Management Tools for Golang
============================================

``emt`` is small a Golang library for managing and aggregating errors. 

Overview
--------

A number of common patterns in Go programming, like thread pools,
producer/consumer flows and anything with "continue on error" semantics, can
become bulky and awkward in light of Go's approach to error handling. In years
of writing production Go code, I have reached for earlier versions of the
types in this package, or cobbled together simple facsimiles more times than I
care to remember. 

Re-implementing error aggregation tools, can be delicate and subtle bugs can
lurk until maximally inopportune times. This package is an attempt to provide
this general utility function, with the hope that it will facilitate more
productive development, and improve the readability of higher level code.

Functionality
-------------

See the full `go documentation
<https://pkg.go.dev/github.com/tychoish/emt>`_ for complete information.

Catcher
~~~~~~~

The ``Catcher`` interface collects errors, potentially from multiple threads,
and supports workflows for continue-on-error semantics for
worker-pools. Imagine the following: :: 

    wg := &sync.WaitGroup{}
    catcher := emt.NewCatcher()
    for i := 0; i <= 10; i++ {
        wg.Add(1)
	go func() {
            defer wg.Done()

            err := doRequest(<...>)

	    catcher.Add(err)
	}()
    }
    
    wg.Wait()

    err := catcher.Resolve()

    if err != nil {
        return fmt.Errorf("request pool encountered error: %w", err)
    }

This kind of flow allows you to capture and propagate errors without adding an
error channel (that might have capcity problems,) and it's own
consumer/aggregation functionality, or implement a type substantially similar
to the error catcher.

To simplify calling code the ``Catcher`` interface provides several additional
methods and paradims: 

- the ``Check<>`` methods allow catchers to consume and execute simple check
  functions (``func() error``) which has proven useful for validation code.

- the ``<>When`` methods can improve the readability of calling code, that
  involves "if x, add error". 
  
- the ``<>Extend`` methods make it possible to consume slices of errors, for
  integration with existing error aggreation code.

- The ``Errorf`` and ``New`` methods make it possible to avid nesting calls to
  ``errors.New`` or ``fmt.Errorf``. 

In addition, to the basic implementations--which differ only on the way that
the constituent errors are converted to strings--a "timestamp" catcher
implementation exists, which annotates each error with the time it was
collected, to improve the intelligibility of errors collected over a long
period of time.

Error Channel
~~~~~~~~~~~~~

While the ``Catcher`` provides a great deal of flexibility in a number of
cases, it's approach (provide a light weight structure guarded by mutexes) can
make it more awkward in some situations that make heavier use of channels and
more event-driven code, where you want to consume errors as they're produced.

The ``ErrorChannel`` type wraps two channels for input and output, but also
provides ``Collect`` which takes a context and blocks appropriately, and a
``Resolve`` method that allows the object to behave like a ``Catcher``. When
the context passed on construction is canceled or the ``Stop`` method is
called, incoming errors will no longer be processed. 

By separating the input and output channels, and avoiding closing either
channel, this interface removes some of the sharp corners from plain ``Chan
error`` alternatives.

Objectives
----------

- ``emt`` should require no additional dependencies beyond the standard
  library, and should strive to be an easy for users to adopt and maintain.

- All exported types should be safe for concurrent use from multiple go
  routines without additional concurrency control.

- No panics should originate in the ``emt`` package.

Development
-----------

``emt`` is licensed under the terms of the Apache v2 License, contributions
are welcome: please open issues or pull requests.

