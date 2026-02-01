# Check current go.mod
cat go.mod | head -10

# Fix all the problematic dependencies
sed -i 's/golang.org\/x\/crypto v0.45.0/golang.org\/x\/crypto v0.23.0/' go.mod
sed -i 's/golang.org\/x\/net v0.47.0/golang.org\/x\/net v0.25.0/' go.mod
sed -i 's/golang.org\/x\/sys v0.38.0/golang.org\/x\/sys v0.20.0/' go.mod
sed -i 's/golang.org\/x\/text v0.31.0/golang.org\/x\/text v0.15.0/' go.mod
sed -i 's/golang.org\/x\/tools v0.38.0/golang.org\/x\/tools v0.21.0/' go.mod
sed -i 's/golang.org\/x\/mod v0.29.0/golang.org\/x\/mod v0.17.0/' go.mod
sed -i 's/golang.org\/x\/sync v0.18.0/golang.org\/x\/sync v0.7.0/' go.mod
sed -i '/^toolchain/d' go.mod

# Remove old go.sum
rm -f go.sum

# Regenerate go.sum
docker run --rm -v $(pwd):/app -w /app golang:1.23-alpine sh -c "apk add --no-cache git gcc musl-dev && go mod tidy"

# Verify go.sum was created
ls -lh go.sum

# Now build
docker-compose up -d --build
