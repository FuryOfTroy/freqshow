document.addEventListener('DOMContentLoaded', () => {
    const runEqButton = document.getElementById('runEqButton');
    const outputLog = document.getElementById('outputLog');

    // Initially disable the button until the Go backend is ready
    runEqButton.disabled = true;
    outputLog.textContent = 'Initializing Go backend...';

    const checkGoBackend = setInterval(() => {
        if (typeof window.RunEQCommand !== 'undefined') {
            clearInterval(checkGoBackend); // Stop checking once RunEQCommand is available
            runEqButton.disabled = false; // Enable the button
            outputLog.textContent = 'Go backend ready. Enter parameters and click Run EQ.';
            console.log('Go backend (window.RunEQCommand) is ready.');

            runEqButton.addEventListener('click', async () => {
                const inputFile = document.getElementById('inputFile').value;
                const outputFile = document.getElementById('outputFile').value;
                const freqStart = parseFloat(document.getElementById('freqStart').value);
                const freqEnd = parseFloat(document.getElementById('freqEnd').value);
                const gain = parseFloat(document.getElementById('gain').value);

                outputLog.textContent = 'Running EQ command... Please wait.';
                console.log('Calling Go RunEQCommand with:', { inputFile, outputFile, freqStart, freqEnd, gain });

                try {
                    // Call the Go function exposed via webview.Bind()
                    // The arguments must match the Go function signature
                    const error = await window.RunEQCommand(inputFile, outputFile, freqStart, freqEnd, gain);

                    if (error) {
                        // If the Go function returns an error, it will be caught here
                        // and 'error' will be a JavaScript Error object
                        outputLog.textContent = `Error running EQ command:\n${error.message}`;
                        console.error('Go function returned an error:', error);
                    } else {
                        outputLog.textContent = `EQ command finished successfully. Output saved to ${outputFile}`;
                        console.log('Go function completed successfully.');
                    }
                } catch (e) {
                    outputLog.textContent = `An unexpected error occurred:\n${e.message || e}`;
                    console.error('Unexpected error during Go function call:', e);
                }
            });
        } else {
            console.log('Waiting for window.RunEQCommand to be defined...');
        }
    }, 100); // Check every 100 milliseconds
});
