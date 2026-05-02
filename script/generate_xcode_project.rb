#!/usr/bin/env ruby
# frozen_string_literal: true

require 'fileutils'
require 'xcodeproj'

repo_root = File.expand_path('..', __dir__)
app_dir = File.join(repo_root, 'apps', 'LarkFSDesktop')
project_path = File.join(app_dir, 'LarkFSDesktop.xcodeproj')

FileUtils.rm_rf(project_path)

project = Xcodeproj::Project.new(project_path)
project.main_group.path = '.'
project.main_group.source_tree = '<group>'

app_target = project.new_target(:application, 'LarkFSDesktop', :osx, '14.0')
extension_target = project.new_target(:app_extension, 'LarkFSFileProviderExtension', :osx, '14.0')

def set_common_build_settings(target)
  target.build_configurations.each do |config|
    settings = config.build_settings
    settings['CLANG_ENABLE_MODULES'] = 'YES'
    settings['DEVELOPMENT_TEAM'] = ''
    settings['ENABLE_USER_SCRIPT_SANDBOXING'] = 'NO'
    settings['MACOSX_DEPLOYMENT_TARGET'] = '14.0'
    settings['SDKROOT'] = 'macosx'
    settings['SWIFT_VERSION'] = '6.0'
    settings['SWIFT_STRICT_CONCURRENCY'] = 'complete'
  end
end

set_common_build_settings(app_target)
set_common_build_settings(extension_target)

app_target.build_configurations.each do |config|
  settings = config.build_settings
  settings['CODE_SIGN_ENTITLEMENTS'] = 'App/LarkFSDesktop.entitlements'
  settings['GENERATE_INFOPLIST_FILE'] = 'NO'
  settings['INFOPLIST_FILE'] = 'App/Info.plist'
  settings['LD_RUNPATH_SEARCH_PATHS'] = '$(inherited) @executable_path/../Frameworks'
  settings['PRODUCT_BUNDLE_IDENTIFIER'] = 'dev.ichen.larkfs.desktop'
  settings['PRODUCT_NAME'] = 'LarkFSDesktop'
end

extension_target.build_configurations.each do |config|
  settings = config.build_settings
  settings['APPLICATION_EXTENSION_API_ONLY'] = 'YES'
  settings['CODE_SIGN_ENTITLEMENTS'] = 'FileProviderSupport/LarkFSFileProvider.entitlements'
  settings['GENERATE_INFOPLIST_FILE'] = 'NO'
  settings['INFOPLIST_FILE'] = 'FileProviderSupport/Info.plist'
  settings['LD_RUNPATH_SEARCH_PATHS'] = '$(inherited) @executable_path/../Frameworks @executable_path/../../../../Frameworks'
  settings['PRODUCT_BUNDLE_IDENTIFIER'] = 'dev.ichen.larkfs.desktop.fileprovider'
  settings['PRODUCT_NAME'] = 'LarkFSFileProvider'
  settings['SKIP_INSTALL'] = 'YES'
end

def add_sources(project, target, paths)
  paths.each do |relative_path|
    file = project.main_group.find_file_by_path(relative_path) || project.main_group.new_file(relative_path)
    target.source_build_phase.add_file_reference(file)
  end
end

def add_resources(project, target, paths)
  paths.each do |relative_path|
    file = project.main_group.find_file_by_path(relative_path) || project.main_group.new_file(relative_path)
    target.resources_build_phase.add_file_reference(file)
  end
end

def add_larkfs_bridge_phase(target)
  phase = target.new_shell_script_build_phase('Copy larkfs bridge')
  phase.always_out_of_date = '1' if phase.respond_to?(:always_out_of_date=)
  phase.output_paths = ['$(TARGET_BUILD_DIR)/$(UNLOCALIZED_RESOURCES_FOLDER_PATH)/bin/larkfs']
  phase.shell_script = <<~'SH'
    set -euo pipefail

    ROOT_DIR="$(cd "$SRCROOT/../.." && pwd)"
    BRIDGE="$ROOT_DIR/bin/larkfs"
    DEST_DIR="$TARGET_BUILD_DIR/$UNLOCALIZED_RESOURCES_FOLDER_PATH/bin"

    mkdir -p "$ROOT_DIR/bin" "$DEST_DIR"
    (cd "$ROOT_DIR" && go build -o "$BRIDGE" ./cmd/larkfs)

    cp "$BRIDGE" "$DEST_DIR/larkfs"
    chmod +x "$DEST_DIR/larkfs"
  SH
end

native_sources = Dir.chdir(app_dir) { Dir['NativeBridge/**/*.swift'].sort }
app_sources = Dir.chdir(app_dir) do
  Dir['App/**/*.swift', 'Models/**/*.swift', 'Services/**/*.swift', 'Stores/**/*.swift', 'Support/**/*.swift', 'Views/**/*.swift'].sort
end
extension_sources = Dir.chdir(app_dir) { Dir['FileProviderExtension/**/*.swift'].sort }

add_sources(project, app_target, app_sources + native_sources)
add_sources(project, extension_target, extension_sources + native_sources)
add_resources(project, app_target, [
  'Resources/AppIcon.icns',
  'Resources/AppIconDark.icns',
  'Resources/AppIconLight.png',
  'Resources/AppIconDark.png'
])
add_larkfs_bridge_phase(extension_target)
add_larkfs_bridge_phase(app_target)

app_target.add_dependency(extension_target)
embed_phase = app_target.new_copy_files_build_phase('Embed App Extensions')
embed_phase.dst_subfolder_spec = '13'
build_file = embed_phase.add_file_reference(extension_target.product_reference, true)
build_file.settings = { 'ATTRIBUTES' => ['RemoveHeadersOnCopy'] }

scheme = Xcodeproj::XCScheme.new
scheme.add_build_target(app_target)
scheme.set_launch_target(app_target)
scheme.save_as(project_path, 'LarkFSDesktop', true)

extension_scheme = Xcodeproj::XCScheme.new
extension_scheme.add_build_target(extension_target)
extension_scheme.save_as(project_path, 'LarkFSFileProviderExtension', true)

project.save
