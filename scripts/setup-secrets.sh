#!/bin/bash
# Helper script to generate and validate GitHub Secrets for Afarias19
# Usage: ./scripts/setup-secrets.sh [generate|validate|check]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
log_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# ============================================================================
# GPG Setup
# ============================================================================

generate_gpg_key() {
    log_info "Generating GPG key for code signing..."
    
    if ! command_exists gpg; then
        log_error "GPG not installed. Install with: brew install gnupg"
        return 1
    fi
    
    local key_name="Afarias19 Release Bot"
    local key_email="releases@afarias19.local"
    local key_expiry="3y"
    
    log_info "Creating GPG key with:"
    echo "  Name: $key_name"
    echo "  Email: $key_email"
    echo "  Expiry: $key_expiry"
    echo ""
    
    gpg --list-secret-keys --keyid-format long | grep -A 1 "sec" || true
    
    log_success "GPG key generation steps:"
    echo "1. Run: gpg --full-generate-key"
    echo "2. Select RSA (4096 bits)"
    echo "3. Set name: $key_name"
    echo "4. Set email: $key_email"
    echo "5. Set expiry: $key_expiry"
}

export_gpg_private_key() {
    log_info "Exporting GPG private key..."
    
    if ! command_exists gpg; then
        log_error "GPG not installed"
        return 1
    fi
    
    local key_id="${1:-}"
    
    if [ -z "$key_id" ]; then
        log_info "Available GPG keys:"
        gpg --list-secret-keys --keyid-format long | grep "sec" || log_warning "No keys found"
        log_info "Usage: $0 export-gpg-key [KEY_ID]"
        return 1
    fi
    
    local export_file="$PROJECT_ROOT/.secrets/gpg-private-key.asc"
    mkdir -p "$PROJECT_ROOT/.secrets"
    
    gpg --armor --export-secret-keys "$key_id" > "$export_file"
    chmod 600 "$export_file"
    
    log_success "Private key exported to: $export_file"
    
    # Encode as base64
    local b64_file="$export_file.b64"
    base64 "$export_file" > "$b64_file"
    
    log_info "Base64 encoded for GitHub Secret: GPG_PRIVATE_KEY"
    echo ""
    echo "=== First 300 chars of GPG_PRIVATE_KEY ==="
    head -c 300 "$b64_file"
    echo ""
}

# ============================================================================
# Apple Setup
# ============================================================================

export_apple_p12() {
    log_info "Apple Developer ID P12 export instructions:"
    echo ""
    echo "1. Open Keychain Access"
    echo "2. Find certificate: 'Developer ID Application: [Your Name]'"
    echo "3. Right-click → Export"
    echo "4. Save as: DeveloperID.p12"
    echo "5. Enter a strong password"
    echo ""
    
    # Check if file exists
    if [ -f "$PROJECT_ROOT/DeveloperID.p12" ]; then
        log_success "DeveloperID.p12 found"
        
        local b64_file="$PROJECT_ROOT/.secrets/apple-p12.b64"
        mkdir -p "$PROJECT_ROOT/.secrets"
        
        base64 "$PROJECT_ROOT/DeveloperID.p12" > "$b64_file"
        
        log_info "Base64 encoded for GitHub Secret: APPLE_DEVELOPER_ID_P12"
        echo ""
        echo "=== First 300 chars of APPLE_DEVELOPER_ID_P12 ==="
        head -c 300 "$b64_file"
        echo ""
    else
        log_warning "DeveloperID.p12 not found"
        log_info "Save the P12 file to: $PROJECT_ROOT/DeveloperID.p12"
    fi
}

# ============================================================================
# DigiCert Setup
# ============================================================================

digicert_instructions() {
    log_info "DigiCert setup instructions:"
    echo ""
    echo "1. Visit: https://www.digicert.com/"
    echo "2. Sign in to DigiCert dashboard"
    echo "3. Navigate to: API or Code Signing section"
    echo "4. Note down:"
    echo "   - DIGICERT_CLIENT_ID"
    echo "   - DIGICERT_CLIENT_SECRET"
    echo "   - DIGICERT_CERTIFICATE_SHA1"
    echo ""
    log_warning "Add these to GitHub Secrets!"
}

# ============================================================================
# Validation
# ============================================================================

validate_secrets() {
    log_info "Validating GitHub secrets..."
    
    local secrets=(
        "DIGICERT_CLIENT_ID"
        "DIGICERT_CLIENT_SECRET"
        "DIGICERT_CERTIFICATE_SHA1"
        "APPLE_DEVELOPER_ID_P12"
        "APPLE_DEVELOPER_ID_PASSWORD"
        "APPLE_DEVELOPER_ID_NAME"
        "APPLE_TEAM_ID"
        "APPLE_NOTARIZE_PASSWORD"
        "GPG_PRIVATE_KEY"
        "GPG_PASSPHRASE"
    )
    
    if ! command_exists gh; then
        log_error "GitHub CLI not installed. Install: brew install gh"
        return 1
    fi
    
    if ! gh auth status >/dev/null 2>&1; then
        log_error "Not authenticated with GitHub. Run: gh auth login"
        return 1
    fi
    
    for secret in "${secrets[@]}"; do
        if gh secret list | grep -q "^$secret"; then
            log_success "Configured: $secret"
        else
            log_warning "Missing: $secret"
        fi
    done
}

# ============================================================================
# Main
# ============================================================================

main() {
    local command="${1:-help}"
    
    case "$command" in
        generate-gpg)
            generate_gpg_key
            ;;
        export-gpg-key)
            export_gpg_private_key "$2"
            ;;
        export-apple)
            export_apple_p12
            ;;
        digicert)
            digicert_instructions
            ;;
        validate)
            validate_secrets
            ;;
        help|*)
            echo "Afarias19 Secrets Setup Helper"
            echo ""
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  generate-gpg         Generate GPG key for signing"
            echo "  export-gpg-key ID    Export GPG key as base64"
            echo "  export-apple         Export Apple P12 certificate"
            echo "  digicert             DigiCert setup instructions"
            echo "  validate             Check all GitHub secrets"
            echo "  help                 Show this help"
            ;;
    esac
}

main "$@"
