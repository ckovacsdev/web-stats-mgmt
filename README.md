# web-stats-mgmt

A state-less Go microservice to get stats about a website, user provides google api token and url.
Not yet complete

TODO:
[] Upgrade Go and Fiber to latest versions
[] Add Rate Limiting
[] Add CORS (if only going to be used on my website), and restrict URLs if meant to keep for my site only
[] Decide deployment and secrets management strategy
[] Verify response size + match cap accordingly
[] Look into caching results / CRON job since requests take 10-20s and want data to be ready on site load 
[] Potentially look into logging solution
