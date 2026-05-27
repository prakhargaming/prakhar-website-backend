# Golang backend for `prakhargaming.com`.

I wanted to migrate the backend of `prakhargaming.com` away from a serverless architecture and more towards a traditional backend written in Golang. 
The backend terminates at `https://api.prakhargaming.com` and it's super awesome. Here are some details:

Deployed a production Go backend to AWS EC2 with TLS-terminating Caddy reverse proxy with:
- systemd hardening
- in-memory token-bucket rate limiting
- and optional Clerk JWT auth

The backend mainly serves a chat endpoint backed by MongoDB Atlas vector search and Google Gemini.
