# Golang backend for `prakhargaming.com`

I wanted to migrate the backend of `prakhargaming.com` away from a serverless architecture and more towards a traditional backend written in Golang. 
The backend terminates at `https://api.prakhargaming.com` and it's super awesome. Here are some details:

This is a production Go backend deployed to AWS EC2 with TLS-terminating Caddy reverse proxy. Additional features include:
- systemd hardening
- in-memory token-bucket rate limiting
- optional Clerk JWT auth
- and blog serving from MongoDB.
