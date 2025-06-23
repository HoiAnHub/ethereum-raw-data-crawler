# 🚀 Quick Start - Ensuring Latest Code

## 🔥 IMPORTANT: Always Use Fresh Builds for Code Changes

When you make code changes, Docker cache can prevent your changes from being built. **Always use these commands to ensure latest code:**

## ⚡ Recommended Commands

### For Development (Code Changes)
```bash
# 🔥 ALWAYS use this when you change code
make scheduler-up-fresh

# OR use the deploy script
./scripts/deploy.sh fresh
```

### For Production Deployment
```bash
# Production with fresh build
make deploy-production

# OR use the deploy script
./scripts/deploy.sh prod
```

### When Having Docker Cache Issues
```bash
# Nuclear option - clean everything
make docker-clean-build && make scheduler-up

# OR use the deploy script
./scripts/deploy.sh clean
```

## 🔍 Verify Your Deployment

After deployment, **always verify** your environment:

```bash
# Quick check
make env-check-container

# Comprehensive check
make env-check-full

# OR use the deploy script
./scripts/deploy.sh check
```

## 📋 Common Workflows

### Development Workflow
```bash
# 1. Make code changes
# 2. Force rebuild and start
make scheduler-up-fresh

# 3. Check environment
make env-check-container

# 4. View logs
make scheduler-logs
```

### Production Workflow
```bash
# 1. Test in development first
./scripts/deploy.sh fresh

# 2. Deploy to production
./scripts/deploy.sh prod

# 3. Verify deployment
./scripts/deploy.sh check
```

### Troubleshooting Workflow
```bash
# 1. Stop everything
make scheduler-down

# 2. Clean Docker cache
make docker-clean-build

# 3. Start fresh
make scheduler-up

# 4. Check status
make env-check-full
```

## ⚠️ Common Mistakes to Avoid

### ❌ DON'T DO THIS:
```bash
# This might use cached/old code
make scheduler-up
```

### ✅ DO THIS INSTEAD:
```bash
# This ensures latest code
make scheduler-up-fresh
```

## 🛠️ All Available Commands

### Makefile Commands
```bash
make scheduler-up-fresh      # Force rebuild and start (RECOMMENDED)
make docker-build-fresh      # Build image with no cache
make docker-clean-build      # Clean and rebuild everything
make deploy-production       # Production deployment
make env-check-container     # Check container environment
make env-check-full          # Full environment check with tests
```

### Deploy Script Commands
```bash
./scripts/deploy.sh fresh    # Fresh development build
./scripts/deploy.sh prod     # Production deployment
./scripts/deploy.sh clean    # Clean everything and rebuild
./scripts/deploy.sh check    # Check status and environment
./scripts/deploy.sh logs     # Show logs
./scripts/deploy.sh stop     # Stop services
```

## 🚨 Remember

- **Always use `fresh` commands when you change code**
- **Verify your deployment after changes**
- **Use production commands for actual deployment**
- **Clean build if you have Docker cache issues**

## 🔗 Need More Help?

- Full documentation: [README.md](README.md)
- Environment setup: [env.example](env.example)
- Production deployment: [.env.scheduler.example](.env.scheduler.example)