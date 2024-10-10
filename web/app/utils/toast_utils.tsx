import { Alert, AlertActionCloseButton, Button, Flex, FlexItem } from '@patternfly/react-core';
import React from 'react';
import { toast } from 'react-hot-toast';

/**
 * Return a <div> containing a <Flex> to be used as a toast notification.
 * @param header Name or title of the error
 * @param body Error message
 * @param variant The variant of alert to display
 * @param dismissToast Called when the toast should be dismissed.
 */
export function GetToastContentWithHeaderAndBody(
    header: string,
    body: string,
    variant: 'danger' | 'warning' | 'success' | 'info' | 'custom',
    dismissToast: () => void,
): React.JSX.Element {
    return (
        <Alert
            isInline
            variant={variant}
            title={header}
            timeoutAnimation={30000}
            timeout={10000}
            onTimeout={() => dismissToast()}
            actionClose={<AlertActionCloseButton onClose={() => dismissToast()} />}
        >
            <p>{body}</p>
        </Alert>
    );
}

export async function ToastFetch(
    loadingMessage: string,
    successToast: (toastId: string) => React.JSX.Element,
    errorToast: (resp: Response, reason: string, toastId: string) => React.JSX.Element,
    endpoint: string,
    requestOptions: any,
) {
    const toastId: string = toast.loading(loadingMessage);
    await fetch(endpoint, requestOptions).then((res) => {
        if (!res.ok || res.status >= 300) {
            res.json().then((reason) => {
                console.error(`HTTP ${res.status} - ${res.statusText}: ${JSON.stringify(reason)}`);
                toast.error(errorToast(res, reason, toastId), {
                    id: toastId,
                    duration: 10000,
                    style: { maxWidth: 600 },
                });
            });
        } else {
            toast.success(
                () => (
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsNone' }}
                        align={{ default: 'alignRight' }}
                        alignContent={{ default: 'alignContentFlexEnd' }}
                        justifyContent={{ default: 'justifyContentFlexEnd' }}
                    >
                        <FlexItem>{successToast(toastId)}</FlexItem>
                        <FlexItem
                            spacer={{ default: 'spacerNone' }}
                            align={{ default: 'alignRight' }}
                            alignSelf={{ default: 'alignSelfFlexEnd' }}
                        >
                            <Button
                                variant={'link'}
                                onClick={() => {
                                    toast.dismiss(toastId);
                                }}
                            >
                                Dismiss
                            </Button>
                        </FlexItem>
                    </Flex>
                ),
                { id: toastId, duration: 7500, style: { maxWidth: 600 } },
            );
        }
    });
}
