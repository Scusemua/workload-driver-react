import React from 'react';
import { ChangeHandler, CodeEditor, Language } from '@patternfly/react-code-editor';
import { Chip, Grid, GridItem } from '@patternfly/react-core';

export interface CodeEditorComponent {
    children?: React.ReactNode;
    onChange?: ChangeHandler;
}

export const CodeEditorComponent: React.FunctionComponent<CodeEditorComponent> = (props) => {
    const onEditorDidMount = (editor, monaco) => {
        editor.layout();
        editor.focus();
        monaco.editor.getModels()[0].updateOptions({ tabSize: 5 });
    };

    const shortcuts = [
        {
            keys: ['Opt', 'F1'],
            description: 'Accessibility helps',
        },
        {
            keys: ['F1'],
            description: 'View all editor shortcuts',
        },
        {
            keys: ['Ctrl', 'Space'],
            description: 'Activate auto complete',
        },
        {
            keys: ['Cmd', 'S'],
            description: 'Save',
        },
    ];
    const shortcutsPopoverProps = {
        bodyContent: (
            <Grid span={6} hasGutter key="grid">
                {shortcuts.map((shortcut, index) => (
                    <React.Fragment key={index}>
                        <GridItem style={{ textAlign: 'right', marginRight: '1em' }}>
                            {shortcut.keys
                                .map((key) => (
                                    <Chip key={key} isReadOnly>
                                        {key}
                                    </Chip>
                                ))
                                .reduce((prev, curr) => (
                                    <>{[prev, ' + ', curr]}</>
                                ))}
                        </GridItem>
                        <GridItem>{shortcut.description}</GridItem>
                    </React.Fragment>
                ))}
            </Grid>
        ),
        'aria-label': 'Shortcuts',
    };

    return (
        <CodeEditor
            shortcutsPopoverProps={shortcutsPopoverProps}
            isLanguageLabelVisible
            isUploadEnabled
            isDownloadEnabled
            isCopyEnabled
            onChange={props.onChange}
            language={Language.python}
            onEditorDidMount={onEditorDidMount}
            height="400px"
        />
    );
};
