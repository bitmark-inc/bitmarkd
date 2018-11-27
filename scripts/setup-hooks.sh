#!/bin/bash

ERROR_CODE=1
SUCCESS_CODE=0

current_path=$(pwd)

# check if this script runs at bitmarkd root directory
if [[ ! "$current_path" == *bitmarkd ]]; then
    printf "\nCurrent directory ${current_path}, please run script at bitmarkd root directory.\n" 
    exit $ERROR_CODE
fi

git_dir="${current_path}/.git"
hook_dir="${git_dir}/hooks"

# backup git hooks directory
if [ -d $hook_dir ] && [ ! -L $hook_dir ]; then
    timestamp=$(date +%s)
    backup_dir="hooks-backup-${timestamp}"
    printf "backup existing hooks directory into ${backup_dir}..."
    mv $hook_dir "${git_dir}/${backup_dir}"
fi

# do nothing if already linked
if [ -d $hook_dir ] && [ -L $hook_dir ]; then
    printf "${hook_dir} already exist, exit..."
    exit $SUCCESS_CODE
fi

# link hook directory
if [ -d "${current_path}/hooks" ]; then
    printf '\nLink hook directory...\n'
    cd $git_dir
    ln -s ../hooks
    cd $current_path
else
    printf "\nError: hooks directory not exist, abort...\n"
    exit $ERROR_CODE
fi
