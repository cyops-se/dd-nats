using System;

namespace OPC.Data
{
    /// <summary>
    /// <see cref="OPC.Data.Interface.IOPCShutdown"/> request event handler
    /// </summary>
    /// <param name="sender"></param>
    /// <param name="e"></param>
    public delegate void ShutdownRequestEventHandler(object sender, ShutdownRequestEventArgs e);

    /// <summary>
    /// <see cref="OPC.Data.Interface.IOPCShutdown"/> request event argument
    /// </summary>
    public class ShutdownRequestEventArgs : EventArgs
    {
        /// <summary>
        /// Shutdown reason
        /// </summary>
        public string ShutdownReason { get; private set; }

        /// <summary>
        /// Shutdown request event argument constructor
        /// </summary>
        /// <param name="shutdownReason"></param>
        public ShutdownRequestEventArgs(string shutdownReason)
        {
            ShutdownReason = shutdownReason;
        }
    }
}
