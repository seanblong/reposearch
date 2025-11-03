# Authentication Setup Guide

## Quick Start (No Authentication)

The application will run in open mode - no login required.

## Enable GitHub Authentication

To enable GitHub OAuth authentication:

### Create GitHub OAuth App

- Go to [https://github.com/settings/applications/new](https://github.com/settings/applications/new)
- Application name: `reposearch`
- Homepage URL: `http://localhost:3000` (or your domain)
- Authorization callback URL: `http://localhost:3000/auth/callback`
- Copy the Client ID and Client Secret

### Configure Environment

TBD
