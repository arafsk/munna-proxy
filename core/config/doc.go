// Package config defines the on-disk configuration schema, validation, and
// secure key storage.
//
// Secrets never live in plain files: keys are stored in the macOS Keychain,
// Windows DPAPI/Credential Manager, Linux Secret Service or the Android
// Keystore. The configuration file only references key identifiers.
//
// TODO(M1): schema, defaults and validation.
// TODO(M2): platform key stores behind a single interface.
// TODO(M6): migrations between schema versions.
package config
