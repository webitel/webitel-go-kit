// Package stdout registers the stdout, stderr and file schemes, one JSON
// record per export. The file DSN is
// file:/path;max-size=100;max-age=30;backups=3;localtime=true;compress=false,
// rotated by lumberjack.
package stdout
