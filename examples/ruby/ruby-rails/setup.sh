#!/usr/bin/env bash
# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

app_dir=${APP_DIR:-demo}

if [ -d "$app_dir" ]; then
    echo "Demo rails app already exists. Please delete $app_dir and re-run make setup"
    exit 1
fi

# Install: https://guides.rubyonrails.org/install_ruby_on_rails.html

mise install ruby@3
mise use -g ruby@3

mise exec ruby@3 -- ruby --version

mise exec ruby@3 -- gem install rails

mise exec ruby@3 -- rails --version

# Generate: https://guides.rubyonrails.org/getting_started.html

mise exec ruby@3 -- rails new "$app_dir"

cd "$app_dir"

mise exec ruby@3 -- bundle remove image_processing

sed -i.bak '/config.load_defaults/a\
    config.active_storage.variant_processor = :disabled
' config/application.rb
rm config/application.rb.bak

mise exec ruby@3 -- bin/rails generate controller Tasks index --skip-routes --skip-helper --skip-assets

cat > config/routes.rb <<'RUBY'
Rails.application.routes.draw do
  root "tasks#index"
  resources :tasks, only: [:create]
end
RUBY

cat > app/controllers/tasks_controller.rb <<'RUBY'
class TasksController < ApplicationController
  TASKS = []
  TASKS_LOCK = Mutex.new

  def index
    @tasks = TASKS_LOCK.synchronize { TASKS.dup }
  end

  def create
    title = params[:title].to_s.strip
    TASKS_LOCK.synchronize { TASKS << title } if title.present?

    redirect_to root_path
  end
end
RUBY

cat > app/views/tasks/index.html.erb <<'ERB'
<main style="max-width: 40rem; margin: 4rem auto; font-family: system-ui, sans-serif;">
  <h1>Tasks</h1>

  <%= form_with url: tasks_path, method: :post do %>
    <%= text_field_tag :title, nil, placeholder: "Add a task", autofocus: true %>
    <%= submit_tag "Add" %>
  <% end %>

  <% if @tasks.any? %>
    <ul>
      <% @tasks.each do |task| %>
        <li><%= task %></li>
      <% end %>
    </ul>
  <% else %>
    <p>No tasks yet.</p>
  <% end %>
</main>
ERB

echo "To run demo:"
echo "mise exec -C $app_dir ruby@3 -- rails server"
