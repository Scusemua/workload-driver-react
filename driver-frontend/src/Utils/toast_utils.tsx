import { RoundToThreeDecimalPlaces } from '@Components/Modals';
import { Alert, AlertActionCloseButton, Button, Flex, FlexItem } from '@patternfly/react-core';
import { SpinnerIcon } from '@patternfly/react-icons';
import React, { ReactElement } from 'react';
import { Toast, toast } from 'react-hot-toast';

/**
 * Return a <div> containing a <Flex> to be used as a toast notification.
 * @param header Name or title of the error
 * @param body Error message
 * @param variant The variant of alert to display
 * @param dismissToast Called when the toast should be dismissed.
 * @param timeout optional timeout for the alert
 */
export function GetToastContentWithHeaderAndBody(
    header: string,
    body: string | ReactElement | (string | ReactElement)[] | undefined,
    variant: 'danger' | 'warning' | 'success' | 'info' | 'custom',
    dismissToast: () => void,
    timeout: number | undefined = undefined,
): React.JSX.Element {
    const getAlertContent = () => {
        if (!body) {
            return <React.Fragment />;
        }

        if (typeof body === 'string') {
            return <p>{body}</p>;
        }

        if (Array.isArray(body)) {
            return (
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
                    {(body as (string | ReactElement)[]).map((elem: string | ReactElement, index: number) => {
                        if (typeof elem === 'string') {
                            return (
                                <FlexItem key={`toast-content-row-${index}`}>
                                    <p>{elem as string}</p>
                                </FlexItem>
                            );
                        } else {
                            return <FlexItem key={`toast-content-row-${index}`}>{elem as ReactElement}</FlexItem>;
                        }
                    })}
                </Flex>
            );
        }

        throw new Error(`Unexpected type for body parameter: ${body}`);
    };

    return (
        <Alert
            isInline
            variant={variant}
            title={header}
            timeoutAnimation={timeout ? 30000 : undefined}
            timeout={timeout}
            onTimeout={() => dismissToast()}
            actionClose={<AlertActionCloseButton onClose={() => dismissToast()} />}
        >
            {getAlertContent()}
        </Alert>
    );
}

/**
 * Basically an implementation of toast.promise(), but this (a) actually works, and (b) is targeted specifically
 * for refreshing a remote resource.
 *
 * @param refreshFunc the function (usually an SWR hook/trigger/mutator) to perform the refresh. Must be async.
 * @param loadingMessage message to display in toast notification while refreshing.
 * @param errorMessage message to display in toast notification if an error occurs.
 * @param successMessage message to display in toast notification upon successful refresh. this will have " in X milliseconds"
 * appended to the end of it before being displayed.
 */
export function ToastRefresh(
    refreshFunc: () => Promise<any>,
    loadingMessage: string,
    errorMessage: string,
    successMessage: string,
) {
    const toastId: string = toast.custom((t: Toast) => {
        return (
            <Alert
                isInline
                variant={'info'}
                title={loadingMessage}
                onTimeout={() => toast.dismiss(t.id)}
                customIcon={<SpinnerIcon className={'loading-icon-spin-pulse'} />}
                actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
            />
        );
    });

    const start: number = performance.now();
    refreshFunc()
        .then(() => {
            const latencyMs: number = RoundToThreeDecimalPlaces(performance.now() - start);
            toast.custom(
                (t) => {
                    return (
                        <Alert
                            isInline
                            variant={'success'}
                            title={successMessage + ` in ${latencyMs} milliseconds.`}
                            onTimeout={() => toast.dismiss(t.id)}
                            timeout={10000}
                            timeoutAnimation={20000}
                            actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
                        />
                    );
                },
                {
                    id: toastId,
                },
            );
        })
        .catch((err: Error) => {
            console.error(`ToastRefresh ERROR: ${err}`);
            const latencyMs: number = RoundToThreeDecimalPlaces(performance.now() - start);
            toast.custom(
                (t) => {
                    return (
                        <Alert
                            isInline
                            variant={'danger'}
                            title={errorMessage + ` after ${latencyMs} milliseconds`}
                            onTimeout={() => toast.dismiss(t.id)}
                            timeout={30000}
                            timeoutAnimation={60000}
                            actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
                        >
                            {`${err.name}: ${err.message}`}
                        </Alert>
                    );
                },
                { id: toastId },
            );
        });
}

export async function ToastFetch(
    loadingMessage: string,
    successToast: (toastId: string) => React.JSX.Element,
    errorToast: (resp: Response, reason: string, toastId: string) => React.JSX.Element,
    endpoint: string,
    requestOptions: RequestInit | undefined,
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
