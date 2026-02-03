#!/bin/sh
set -e
fly volumes create mouse_data --size 10 --region lhr
