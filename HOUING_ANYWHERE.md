# Solution comments

I'd like to explain a few decisions I made in this project as it may not be obvious from reading the code  

## No Docker Image

Since this project is not a REST service but rather a command-line tool meant to be executed
on a desktop along with Anki app, it doesn't make any sense to pack it in a Docker image as it
would require ``--network host`` flag. 
This flag doesn't work fine on Mac and requires a workaround, so it would be very inconvenient.

On the other hand, Go natively supports cross-compilation with great variety of target platforms,
so it's very easy to get a binary executable for any platform. 

Thus, I just provided instruction on how to build the tool using Docker without installing Go locally.

## Hand-Made Templates

It may look very wierd that I chose to write my own variable-substitution function 
instead of using standard [text/template](https://pkg.go.dev/text/template) package.
This is a very doubtful indeed, and if had been writing a production system, 
I would have definitely used the standard templates.

However, this is a pet project, so I decided to write my own function for the following reasons:

1. Standard templates are notorious for non-informative error messages. I wanted to have more control over the
   errors to simplify debugging -- it's much easier to debug 60 LOC written by myself than to debug a huge standard package.
2. I just wanted to have fun! This is a pet project!

# Assignment Questions

There were a few questions in the assignment description. 
I'd like to answer them here as if I wrote the Drone Navigation Service. 

## What instrumentation this service would need to ensure its observability and operational transparency?

I would do standard REST service monitoring for this service. To do so, the service should export
the following metrics:

- OS-related: CPU usage, RAM usage
- Go-specific: GC pause length histogram
- Standard REST metrics: request processing latency histogram, 
  per-status code response counts.

## Why throttling is useful (if it is)? How would you implement it here?

This service doesn't do any network communication with other services or databases,
so the heaviest part of request processing is actually HTTP request parsing and response writing.

Thus, throttling at application level doesn't seem to have any effect. If we reach the service throughput,
I believe we'd better apply one of the following:

- if the drones request DNS service at a fixed rate creating load spikes, adding random delay on the client side may help
  eliminate these spikes.
- otherwise, simply deploy multiple DNS services per Sector and configure drones to access random service.
  our service is stateless and self-sufficient, so there is no problem in replicating it.

If we don't change code on the drones, then throttling should be done at network level by limiting
the TCP pending connections queue and limiting the number of concurrently served requests by the HTTP server.

## What we have to change to make DNS be able to service several sectors at the same time?

Either drones should know SectorID they operate in and attach it with the request, 
or we should implement Sector determination logic based on the coordinates.

## Integration with MomCorp software

The easiest thing that comes to mind is to create a separate endpoint for MomCorp and serve data in the proper form
If it's not good for some reason, we can implement Content-Negotiation logic -- MomCorp ships would provide
some special Content-Type in their ``Accept`` header. Based on that special type, 
we would choose proper data presentation format.

On the backend side, we would use layered architecture, so location computation logic is does not depend on
REST transfer format. So we would continue calling this function and wrap the result with the appropriate response structure.

## How would you enable scenario where DNS can serve both types of clients?

The answer is quite similar to the previous one -- at the endpoint level we should determine the type of the client
and choose which logic to use.

On the other hand, it may be a good idea to deploy a separate next-gen navigation service for new drones.
This way, we reduce the legacy burden from the previous generation of drones. On the other hand, any bugs found
in the DNS should be back-ported to older installations, which may be a huge pain in the neck. 

## How would you separate technical decision to deploy something from business decision to release something?

I would definitely like to be able to enable and disable certain features without redeploying a service.
Ideally, there should be a way to gradually roll out a feature to users and perform A/B testing.
This way, we deploy a new version of our service with new feature disabled, and once the business decision to release is made,
we start a slow roll-out (or just enable the feature instantly, depending on the situation).


