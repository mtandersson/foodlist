import {describe, it, expect, vi} from "vitest"
import {render, screen, fireEvent} from "@testing-library/svelte"
import AboutModal from "./AboutModal.svelte"

describe("AboutModal", () => {
  it("should display only client version when server version is null", () => {
    const onCancel = vi.fn()
    render(AboutModal, {
      props: {
        clientVersion: "1.6.0-dev",
        serverVersion: null,
        onCancel,
      },
    })

    expect(screen.getByText(/Client Version: 1\.6\.0-dev/)).toBeInTheDocument()
    expect(screen.queryByText(/Server:/)).not.toBeInTheDocument()
  })

  it("should display single version when client and server match", () => {
    const onCancel = vi.fn()
    render(AboutModal, {
      props: {
        clientVersion: "1.6.0",
        serverVersion: "1.6.0",
        onCancel,
      },
    })

    expect(screen.getByText(/Version 1\.6\.0/)).toBeInTheDocument()
    expect(screen.queryByText(/Client:/)).not.toBeInTheDocument()
    expect(screen.queryByText(/Server:/)).not.toBeInTheDocument()
  })

  it("should display both versions when they differ", () => {
    const onCancel = vi.fn()
    render(AboutModal, {
      props: {
        clientVersion: "1.6.0-dev",
        serverVersion: "1.6.0",
        onCancel,
      },
    })

    expect(screen.getByText(/Client: 1\.6\.0-dev/)).toBeInTheDocument()
    expect(screen.getByText(/Server: 1\.6\.0/)).toBeInTheDocument()
  })

  it('should display app name "FoodList"', () => {
    const onCancel = vi.fn()
    render(AboutModal, {
      props: {
        clientVersion: "1.6.0",
        serverVersion: null,
        onCancel,
      },
    })

    expect(screen.getByText("FoodList")).toBeInTheDocument()
  })

  it("should close when backdrop is clicked", () => {
    const onCancel = vi.fn()
    const {container} = render(AboutModal, {
      props: {
        clientVersion: "1.6.0",
        serverVersion: null,
        onCancel,
      },
    })

    const backdrop = container.querySelector(".modal-backdrop")
    expect(backdrop).toBeTruthy()

    if (backdrop) {
      fireEvent.click(backdrop)
      expect(onCancel).toHaveBeenCalledTimes(1)
    }
  })

  it("should close when Escape key is pressed", () => {
    const onCancel = vi.fn()
    render(AboutModal, {
      props: {
        clientVersion: "1.6.0",
        serverVersion: null,
        onCancel,
      },
    })

    fireEvent.keyDown(window, {key: "Escape"})
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it("should close when close button is clicked", () => {
    const onCancel = vi.fn()
    render(AboutModal, {
      props: {
        clientVersion: "1.6.0",
        serverVersion: null,
        onCancel,
      },
    })

    const closeButton = screen.getByLabelText("Stäng")
    fireEvent.click(closeButton)
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it("should close when footer close button is clicked", () => {
    const onCancel = vi.fn()
    render(AboutModal, {
      props: {
        clientVersion: "1.6.0",
        serverVersion: null,
        onCancel,
      },
    })

    const footerButton = screen.getByText("Stäng")
    fireEvent.click(footerButton)
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it("should handle undefined server version", () => {
    const onCancel = vi.fn()
    render(AboutModal, {
      props: {
        clientVersion: "1.6.0-dev",
        serverVersion: undefined as any,
        onCancel,
      },
    })

    expect(screen.getByText(/Client Version: 1\.6\.0-dev/)).toBeInTheDocument()
    expect(screen.queryByText(/Server:/)).not.toBeInTheDocument()
  })
})
