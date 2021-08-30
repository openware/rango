# frozen_string_literal: true

require 'rubygems'
require 'bundler'
Bundler.setup(:deploy)

# Load DSL and Setup Up Stages
require 'capistrano/setup'

# Includes default deployment tasks
require 'capistrano/deploy'

require 'capistrano/scm/git'
install_plugin Capistrano::SCM::Git

require 'capistrano/shell'
# require 'capistrano/bundler'
require 'slackistrano/capistrano'
require 'capistrano/goenv'
require 'capistrano/my'
install_plugin Capistrano::My
require 'capistrano/slackistrano' # My Custom Message
# require 'capistrano/dotenv/tasks'
# require 'capistrano/dotenv'
require 'capistrano/systemd/multiservice'
install_plugin Capistrano::Systemd::MultiService.new_service('rango', service_type: 'user')
