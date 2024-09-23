import { Button, Flex, FlexItem, Text, TextVariants } from '@patternfly/react-core';
import React from 'react';
import { toast } from 'react-hot-toast';

/**
 * Return a <div> containing a <Flex> to be used as a toast error notification.
 * @param header Name or title of the error
 * @param body Error message
 */
export function GetHeaderAndBodyForToast(header: string, body: string): React.JSX.Element {
    return (
        <div>
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                <FlexItem>
                    <Text component={TextVariants.p}>
                        <b>{header}</b>
                    </Text>
                </FlexItem>
                <FlexItem>
                    <Text component={TextVariants.small}>{body}</Text>
                </FlexItem>
            </Flex>
        </div>
    );
}

export async function ToastFetch(
    loadingMessage: string,
    successToast: React.JSX.Element,
    errorToast: (resp: Response, reason: string) => React.JSX.Element,
    endpoint: string,
    requestOptions: any,
) {
    const toastId: string = toast.loading(loadingMessage);
    await fetch(endpoint, requestOptions).then((res) => {
        if (!res.ok || res.status >= 300) {
            res.json().then((reason) => {
                console.error(`HTTP ${res.status} - ${res.statusText}: ${JSON.stringify(reason)}`);
                toast.error(errorToast(res, reason), { id: toastId, duration: 10000, style: { maxWidth: 600 } });
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
                        <FlexItem>{successToast}</FlexItem>
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
