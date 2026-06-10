# Pipeline Run Process

## Description

In the glab-pipe system, it should be possible to run new pipelines from the pipeline panel.

## Instructions

- Key `n` starts a new pipeline, opening the instructions modal.
- Key `r` runs the same pipeline again, manually creating a new pipeline.
- Key `u` updates the pipeline panel page, in case it doesn't update automatically.
- Key `Enter` accesses the currently selected pipeline.
- Keyboard arrows and vim commands move between the pipeline list.
- Key `q` closes glab-pipe.
- Key `Esc` returns to the main menu with the list of saved projects.
- Key `c` copies the branch name or accesses the pipeline online.
- Key `s` will stop the current pipeline.

## Visual

- The pipeline list screen should have the following information:
    - Pipeline status
    - Pipeline ID
    - Pipeline branch
    - List of quick keys (Enter, New, Retry, Update, Copy, Quit, Help)
- It should maintain the predefined icon system in the software.

- JetBrains emoji list:

```
 (\uf192) in blue indicating it is running
 (\uf05d) in green indicating it was success
 (\uf52f) in red indicating it was failed
 (\ueabd) in gray indicating it was canceled
 (\uf2be) in orange indicating it was manual
```

## Process

### Starting software

1. First option is using `glab-pipe .` in the local repository of the project on my computer.
2. Second option is using `glab-pipe` anywhere in the terminal.
3. In the first option, the software will start directly in the pipeline screen of the current project.
4. In the second option, it will open the software's initial menu and I must select the project in the main menu.
5. The pipeline screen will be opened.
6. Use the arrows or vim keys to move between pipelines.
7. It should show the last 10 pipelines run or running of the project.

### Running pipelines

1. Pressing `n` goes to the new pipeline creation modal.
2. Pressing `r` will run a new pipeline using the same branch as the selected pipeline.
3. Pressing `u` will update the pipeline screen, if there is a pipeline in Running at the moment the page should update automatically every 2 seconds.
4. When the pipeline status leaves Running it will only update if `u` is pressed.
5. If the pipeline is no longer needed, pressing `s` will cancel the pipeline, showing it as canceled.

### Copying branch name

1. Select one of the pipelines with the arrows or vim commands.
2. Pressing `c` it will ask if:
    - I want to access the pipeline via web, taking the project provider + project path + `-/pipelines/ID`
    - I want to copy the branch name of the selected pipeline.
3. If the first option is selected, it will copy the URL, where I can use Ctrl+V in any browser of my choice.
4. If the second option is selected, it copies the branch name, where I can use Ctrl+V anywhere.

### Accessing jobs

1. In the pipeline screen, select the desired pipeline with the arrows or vim keyboard.
2. Press `Enter` to access the jobs screen of the selected pipeline.


## New pipeline creation modal structure

- It should have white rounded borders.
- It should have a title on the modal border saying `New Pipeline`.
- It will request the Branch name, where it has the following rules:
    - if the branch name starts with CUC, it must put at the beginning of the branch name story/, becoming story/CUC-XXX.
    - if the branch name starts with FY, it must put at the beginning of the branch name release/, becoming release/FYXX_XXXX.
    - if I paste the name copied from the pipeline screen, it should leave it as is without making changes.
    - It should accept any other name after these rules, if some name doesn't follow the previous rules leave it as being written.
- Put the option to be able to add variables as optional, where it can allow putting multiple variables together in the structure `key:value` where each variable is separated by commas.
- Before running the new pipeline, let the user confirm the name and each variable placed in the following form:
    - Confirm the pipeline information
    - Branch name:
    - Variables:
        - Variable1 = value1
        - Variables2 = value2
- Press `Enter` if all data is correct and the system will start the pipeline.
- It should return to the pipeline page where I can see the pipelines, where I should see the new pipeline running.
